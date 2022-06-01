# Decent CLI
Official CLI for the Decent platform.

## Requirements
|Dependency|Version|
|-|-|
|go|1.18+|


## Running
```sh
go run main.go
```


## Commands
|Command|Description|
|-|-|
|`login`|Log in (required to use other commands)|
|`logout`|Log out|
|`auth`|Print current authentication state|
|`init [path?]`|Initialize a new project at the specified path|
|`clone [-p | --path?] [-b | --branch?] [blob]`|Clone a project|
|`changes`|Print local changes|
|`push [-y] [message?]`|Push local changes to remote.|
|`sync [-y] [commit_index?]`|Sync local project to the specified commit (or latest commit if not specified). Retains all local changes unless prompted to override.|
|`reset [-y]`|Reset all local changes to be in sync with remote|
|`revert [-y]`|Revert to the previous commit. **Note: This will also reset all local changes.**|
|`branches`|List all branches in the project|
|`branch new [name]`|Create a new branch|
|`branch use [name]`|Switch to the specified branch for local project|
|`branch delete [-y] [name]`|Delete a branch. **No associated commits or stored project files will be deleted.**|
|`branch set-default [name]`|Set the default branch for the project|
|`history [-l=10]`|Print commit history|
|`status`|Print project config|
### Common flags
|Flag|Description|
|-|-|
|`-y`|Skip confirmation|
|`-l` or `--limit`|Limit returned results|


## Usage

### Ignoring files
Create a `.decentignore` file in your project. Each line will be read as a regular expression (regex),
with all leading and trailing whitespace being ignored. You can also comment out any line with the
`#` prefix.
#### Example
```sh
# Hey look, a comment!
file_name
entire_dir/.*
```
