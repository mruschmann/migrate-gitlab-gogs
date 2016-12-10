package main

import (
	"github.com/xanzy/go-gitlab"
	"github.com/gogits/go-gogs-client"
	"fmt"
	"bufio"
	"os"
	"strconv"
	"strings"
	"encoding/json"
	"io/ioutil"
)


/** The configuration that will be loaded from a JSON file */
type Configuration struct {
	GitlabURL string; ///< URL to the Gitlab API interface
	GitlabAPIKey string; ///< API Key for Gitlab
	GogsURL string; ///< URL to the Gogs API interface
	GogsAPIKey string; ///< API Key for Gogs
	UserMap []UsersMap; ///< Map of Gitlab usernames to Gogs usernames
}


/** An instance of the Configuration to store the loaded data */
var config Configuration


/** The main routine for this program, which migrates a Gitlab project to Gogs
 *
 *  1. Reads the configuration from config.json.
 *  2. Polls the Gitlab server for projects
 *  3. Prompts the user for the Gitlab project to migrate
 *  4. Pools the Gogs server for projects
 *  5. Prompts the user for the Gogs project to migrate
 *  6. Simulates the migration without writing to the Gogs API
 *  7. Prompts the user to press <Enter> to perform the migration
 *  8. Performs the migration of Gitlab project to Gogs Project
 */
func main() {
	var projPtr []*gogs.Repository
	reader := bufio.NewReader(os.Stdin)
	found := false
	num := 0
	var gogsPrj *gogs.Repository
	var gitPrj *gitlab.Project

	// Load configuration from config.json
	file, err9 := ioutil.ReadFile("./config.json")
	CheckError(err9)
	err9 = json.Unmarshal(file, &config)
	fmt.Println("GitlabURL:", config.GitlabURL)
	fmt.Println("GitlabAPIKey:", config.GitlabAPIKey)
	fmt.Println("GogsURL:", config.GogsURL)
	fmt.Println("GogsAPIKey:", config.GogsAPIKey)
	fmt.Println("UserMap: [")
	for i := range config.UserMap {
		fmt.Println("\t", config.UserMap[i].From, "to", config.UserMap[i].To)
	}
	fmt.Println("]")
	CheckError(err9)

	// Have user select a source project from gitlab
	git := gitlab.NewClient(nil, config.GitlabAPIKey)
	git.SetBaseURL(config.GitlabURL)
	opt := &gitlab.ListProjectsOptions{}
	gitlabProjects, _, err := git.Projects.ListProjects(opt)
	CheckError(err)

	fmt.Println("")
	for i := range gitlabProjects {
		fmt.Println(gitlabProjects[i].ID, ":", gitlabProjects[i].Name)
	}

	fmt.Printf("Select source gitlab project: ")
	text, _ := reader.ReadString('\n')
	text = strings.Trim(text, "\n")

	for i := range gitlabProjects {
		num, _ = strconv.Atoi(text)
		if num == gitlabProjects[i].ID {
			found = true
			gitPrj = gitlabProjects[i]
		} // else purposefully omitted
	}
	if !found {
		fmt.Println(text, "not found")
		os.Exit(1)
	} // else purposefully omitted

	// Have user select a destination project in gogs
	gg := gogs.NewClient(config.GogsURL, config.GogsAPIKey)
	projPtr, err = gg.ListMyRepos()
	CheckError(err)

	fmt.Println("")
	for i := range projPtr {
		fmt.Println(projPtr[i].ID, ":", projPtr[i].Name)
	}

	fmt.Printf("Select destination gogs project: ")
	text, _ = reader.ReadString('\n')
	text = strings.Trim(text, "\n")

	for i := range projPtr {
		num, _ = strconv.Atoi(text)
		if int64(num) == projPtr[i].ID {
			found = true
			gogsPrj = projPtr[i]
		} // else purposefully omitted
	}
	if !found {
		fmt.Println(text, "not found")
		os.Exit(1)
	} // else purposefully omitted

	// Perform pre merge
	fmt.Println("\nSimulated migration of", gitPrj.Name, "to", gogsPrj.Name)
	DoMigration(true, git, gg, gitPrj.ID, gogsPrj.Name, gogsPrj.Owner.UserName)

	// Perform actual migration
	fmt.Println("\nCompleted simulation. Press <Enter> to perform migration...")
	text, _ = reader.ReadString('\n')
	DoMigration(false, git, gg, gitPrj.ID, gogsPrj.Name, gogsPrj.Owner.UserName)

	os.Exit(0)
}


/** A map of a milestone from its Gitlab ID to its new Gogs ID */
type MilestoneMap struct {
	from int ///< ID in Gitlab
	to int64 ///< New ID in Gogs
}


/** Performs a migration
 *  \param dryrun Does not write to the Gogs API if true
 *  \param git A gitlab client for making API calls
 *  \param gg A gogs client for making API calls
 *  \param gitPrj ID of the Gitlab project to migrate from
 *  \param gogsPrj The name of the Gitlab project to migrate into
 *  \param owner The owner of gogsPrj, which is required to make API calls
 *
 *  This function migrates the Milestones first. It creates a map from the old
 *  Gitlab milestone IDs to the new Gogs milestone IDs. It uses these IDs to
 *  migrate the issues. For each issue, it migrates all of the comments.
 */
func DoMigration(dryrun bool, git *gitlab.Client, gg *gogs.Client, gitPrj int, gogsPrj string, owner string) {
	var mmap []MilestoneMap
	var listMiles gitlab.ListMilestonesOptions
	var listIssues gitlab.ListProjectIssuesOptions
	var listNotes gitlab.ListIssueNotesOptions
	var err error
	var milestone *gogs.Milestone
	var issueIndex int64
	issueNum := 0
	sort := "asc"

	listIssues.PerPage = 1000
	listIssues.Sort = &sort

	// Migrate all of the milestones
	milestones, _, err0 := git.Milestones.ListMilestones(gitPrj, &listMiles)
	CheckError(err0)
	for i := range milestones {
		fmt.Println("Create Milestone:", milestones[i].Title)

		var opt gogs.CreateMilestoneOption
		opt.Title = milestones[i].Title
		opt.Description = milestones[i].Description
		if !dryrun {
			// Never write to the API during a dryrun
			milestone, err = gg.CreateMilestone(owner, gogsPrj, opt)
			CheckError(err)
			mmap = append(mmap, MilestoneMap{milestones[i].ID, milestone.ID})
		} // else purposefully omitted

		if milestones[i].State == "closed" {
			fmt.Println("Marking as closed")
			var opt2 gogs.EditMilestoneOption
			opt2.Title = opt.Title
			opt2.Description = &opt.Description
			opt2.State = &milestones[i].State
			if !dryrun {
				// Never write to the API during a dryrun
				milestone, err = gg.EditMilestone(owner, gogsPrj, milestone.ID, opt2)
				CheckError(err)
			} // else purposefully omitted
		} // else purposefully omitted
	}

	// Migrate all of the issues
	issues, _, err1 := git.Issues.ListProjectIssues(gitPrj, &listIssues)
	CheckError(err1)
	for i := range issues {
		issueNum++
		if issueNum == issues[i].IID {
			fmt.Println("Create Issue", issues[i].IID, ":", issues[i].Title)

			var opt gogs.CreateIssueOption
			opt.Title = issues[i].Title
			opt.Body = issues[i].Description
			opt.Assignee = MapUser(issues[i].Author.Username) // Gitlab user to Gogs user map
			opt.Milestone = MapMilestone(mmap, issues[i].Milestone.ID)
			opt.Closed = issues[i].State == "closed"
			if !dryrun {
				// Never write to the API during a dryrun
				for k := range issues[i].Labels {
					opt.Labels = append(opt.Labels, GetIssueLabel(git, gg, gitPrj, gogsPrj, owner, issues[i].Labels[k]));
				}
				issue, err6 := gg.CreateIssue(owner, gogsPrj, opt)
				issueIndex = issue.Index
				CheckError(err6)
			} // else purposefully omitted

			// Migrate all of the issue notes
			notes, _, err2 := git.Notes.ListIssueNotes(gitPrj, issues[i].ID, &listNotes)
			CheckError(err2)
			for j := range notes {
				fmt.Println("Adding note", notes[j].ID)

				var opt2 gogs.CreateIssueCommentOption
				//var opt3 gogs.EditIssueCommentOption
				opt2.Body = notes[j].Body
				//opt3.Body = notes[j].Body
				if !dryrun {
					// Never write to the API during a dryrun
					_, err := gg.CreateIssueComment(owner, gogsPrj, issueIndex, opt2)
					//_, err = gg.EditIssueComment(owner, gogsPrj, issueIndex, comment.ID, opt3)
					CheckError(err)
				} // else purposefully omitted
			}
		} else {
			// TODO Create a temp issue and delete it later (MCR 9/29/16)
			fmt.Println("Issues not in order!!")
			fmt.Println("Preservation of skipped issues IDs is not implemented")
			os.Exit(1)
		}
	}
}


/** Find the ID of an label from its name or create a new label
 *  \param git A gitlab client for making API calls
 *  \param gg A gogs client for making API calls
 *  \param gitPrj ID of the Gitlab project to migrate from
 *  \param gogsPrj The name of the Gitlab project to migrate into
 *  \param owner The owner of gogsPrj, which is required to make API calls
 *  \param label The name of the label to find or create
 *  \return The ID of the tag in Gogs
 */
func GetIssueLabel(git *gitlab.Client, gg *gogs.Client, gitPrj int, gogsPrj string, owner string, label string) (int64) {
	ID := int64(-1)
	found := false

	labels, err := gg.ListRepoLabels(owner, gogsPrj)
	CheckError(err)
	for i := range labels {
		if labels[i].Name == label {
			fmt.Println("Found label", label)
			ID = labels[i].ID
			found = true
		}
	}

	if !found {
		tags, _, err1 := git.Labels.ListLabels(gitPrj)
		CheckError(err1)
		for i:= range tags {
			if tags[i].Name == label {
				fmt.Println("Create label", label, "color", tags[i].Color)
				var opt gogs.CreateLabelOption
				opt.Name = label
				opt.Color = tags[i].Color
				tag, err2 := gg.CreateLabel(owner, gogsPrj, opt)
				CheckError(err2)
				found = true
				ID = tag.ID
			}
		}
	} // else purposefully omitted

	if !found {
		fmt.Println("Unable to find label", label, "in gitlab!!")
		os.Exit(5)
	} // else purposefully omitted

	return ID
}


/** An entry in the user map from Gitlab to Gogs */
type UsersMap struct {
	From string ///< The user name to map from
	To string ///< The user name to map to
}


/** Maps a Gitlab user name to the desired Gogs user name
 *  @param user The Gitlab user name to map
 *  @return The Gogs user name
 */
func MapUser(user string) (string) {
	u := user

	for i := range config.UserMap {
		if user == config.UserMap[i].From {
			u = config.UserMap[i].To
		} // else purposefully omitted
	}

	return u
}


/** Maps a Gitlab milestone to the desired Gogs milestone
 *  @param mmap An array of ID maps from Gitlab to Gogs
 *  @param user The Gitlab milestone to map
 *  @return The Gogs milstone
 */
func MapMilestone(mmap []MilestoneMap, ID int) (int64) {
	var toID int64
	toID = int64(ID)

	for i := range mmap {
		if (mmap[i].from == ID) {
			toID = mmap[i].to
		} // else purposefully omitted
	}

	return toID
}


/** Checks an error code and exists if not nil
 *  @param err The error code to check
 */
func CheckError(err error) {
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	} // else purposefully omitted
}
