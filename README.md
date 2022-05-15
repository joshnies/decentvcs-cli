# Quanta Control CLI


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
|`login`|Authenticate with Quanta Control (required to use other commands)|
|`logout`|Log out of Quanta Control|
|`auth`|Print current authentication state|
|`init <path=".">`|Initialize a new project at the specified path|
|`changes`|Print local changes|
|`push <message?> [-y]`|Push local changes to remote.|
|`sync <commit_index?> [-y]`|Sync local project to the specified commit (or latest commit if not specified). Retains all local changes unless prompted to override.|
|`reset [-y]`|Reset all local changes to be in sync with remote|
|`revert [-y]`|Revert to the previous commit. **Note: This will also reset all local changes.**|
|`branches`|List all branches in the project|
|`branch new <name>`|Create a new branch|
|`branch use <name>`|Switch to the specified branch for local project|
|`branch delete <name> [-y]`|Delete a branch. **No associated commits or stored project files will be deleted.**|
|`branch set-default <name>`|Set the default branch for the project|
|`history [-l=10]`|Print commit history|
|`status`|Print project config|
### Common flags
|Flag|Description|
|-|-|
|`-y` or `--no-confirm`|Skip confirmation|
|`-l` or `--limit`|Limit returned results|


## Usage

### Ignoring files
Create a `.qcignore` file in your project. Each line will be read as a regular expression (regex),
with all leading and trailing whitespace being ignored. You can also comment out any line with the
`#` prefix.
#### Example
```sh
# Hey look, a comment!
file_name
entire_dir/.*
```
