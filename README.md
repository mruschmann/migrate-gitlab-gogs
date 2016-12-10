# Gitlab to Gogs Issue Migrator

This is a small app written in go that migrates issues from Gitlab to Gogs. It
uses the Gitlab and Gogs APIs, which results in some limitations. Specifically,
the Gogs API does not permit modification of timestamps.

# What it does

- Migrate issues from Gitlab to Gogs using the APIs
- Migrate issue comments from Gitlab to Gogs
- Migrate Milestones from Gitlab to Gogs
- Create issue labels as necessary
- Use a predefined user map to map Gitlab usernames to Gogs usernames

# What it does not do

- *Preserve timestamps*
- Create or migrate projects
- Create or migrate users
- Migrate the wiki
- Migrate git repositories
- Migrate attachments

# Requirements

- Install go
- Set GOPATH
- *Backup your Gogs data!* Your first migration may not go as planned
- API returns an error if the database cannot be accessed. Therefore, you avoid all other sources of traffic during the migration

# Building and running

Text in single quotations are commands intended to be run on the command line.
Do not include the quotes when you enter them on the command line.

1. Clone this repository
2. Change directory into this repository
3. Run 'go get github.com/plouc/go-gitlab-client'
4. Run 'go get github.com/gogits/go-gogs-client'
5. Run 'go build'
6. Edit config.json
	- Modify the Gitlab API URL to point to your server
	- Change GITLABAPIKEY to your [Gitlab API key](https://www.safaribooksonline.com/library/view/gitlab-cookbook/9781783986842/ch06s05.html)
	- Modify the Gogs API URL to point to your server
	- Change GOGSAPIKEY to your Gogs API key
7. Run ./migrate-gitlab-gogs
8. Enter the number of the Gitlab project that you want to migrate and press
   <enter>
9. Enter the number of the Gogs project that you want to migrate and press
   <enter>
10. Review the simulation information (The script does not attempt to modify the
    Gogs repository during a dry run. Therefore the actual migration may be
    slightly different.)
11. If you are happy with the simulation, then press <Enter> to perform the actual
    migration

After the migration, verify the results in your Gogs repository. If you are not
happy with the migration, then restore your backup and modify this script to
meet your needs.
