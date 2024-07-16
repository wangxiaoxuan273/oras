# oras multi-arch feature design, Version July 15

## PoC

https://github.com/wangxiaoxuan273/oras/pull/1

Only implemented `oras manifest index create`.

## Design
### All commands
`oras manifest index create` Create an image index from source manifests. 

`oras manifest index update` Add/Remove manifests to/from an image index.

### oras manifest index create

Create an image index from source manifests. The command auto-detects platform information for each source manifests. The sources and the target index must be in the same repository.

#### Usage

`oras manifest index create [flags] <name>[:<tag[,<tag>][...]] [{<tag>|<digest>}...]`

#### Flags

(These flags are not implemented in the PoC)

`--subject` Add a subject manifest for the to-be-created index.

`--oci-layout` Set the given repository as an OCI layout.

`--annotation` Add annotations for the to-be-created index.

`--artifact-type` Add artifact type information for the to-be-created index.

(May be added in the future) `--output` Output the updated manifest to a location.

#### Aliases

`pack`

#### Examples

Create an index from source manifests tagged amd64, darwin, armv7 in the repository `localhost:5000/hello`, and push the index without tagging it. We require that amd64, darwin, armv7 exist in `localhost:5000/hello` :
`oras manifest index create localhost:5000/hello amd64 darwin armv7`

Create an index from source manifests tagged amd64, darwin, armv7 in the repository localhost:5000/hello, and push the index with tag `latest`:
`oras manifest index create localhost:5000/hello:latest amd64 darwin armv7`

Create an index from source manifests using both tags and digests, and push the index with tag `latest`:
`oras manifest index create localhost:5000/hello:latest amd64 sha256:xxx armv7`

### oras manifest index update

Add/Remove a manifest from an image index. The updated index will be created as a new index and the old index will not be deleted. 
If the user specify the index with a tag, the corresponding tag will be updated to the new index. If the old index has other tags, the remaining tags will not be updated to the new index.

#### Usage

`oras manifest index update <name>{:<tag>|@<digest>} {--add/--remove} [{<tag>|<digest>}...]`

#### Flags

`--add` (shorthand `-a`) Add a manifest to the index. The manifest will be added as the last element of the index.

`--remove` (shorthand `-r`) Remove a manifest from the index.

**Note: One of the above flags should be used, as there has to be something to update. Otherwise the command does nothing.**

`--oci-layout` Set the target as an oci image layout.

`--tag` Tag the updated index. Multiple tags can be provided.

(May be added in the future) `--output` Output the updated manifest to a location.

#### Examples

Add one manifest and remove two manifests from an index.

`oras manifest index update localhost:5000/hello:latest --add win64 --remove sha256:xxx --remove arm64`

## Issues and considerations
1. **Command design: Making a subcommand group `index` under `oras manifest`(choose `oras manifest index create` instead of `oras manifest create-index` )`**

Reasons:
* The structure of the `oras manifest index` sub command group aligns well with the existing sub command groups `oras manifest/blob/repo`.
* If in the future more index commands are needed, grouping them under the `index` group makes the manifest commands neater. We may also need operations for other manifest types, and creating new sub groups parallel to `index` looks feasible (i.e. `oras manifest image create`).

2. **Combining "add manifest" and "remove manifest" operations as one `index update` operation**

In my past iteration I created separate `oras index add-manifest` and `oras index remove-manifest` commands. After discussion with teammates, I think combining them as one `update` command is a better option, as it makes less garbage and fewer request calls when doing multiple adds and removes. But as a result, the command has many flags and the flags have complicated rules.

3. **`oras manifest index create / update` will auto detect platform information for each source manifest. Currently we don't let user to specify platform unless we have a specific scenario in the future.**

If we allow users to input platform information (introducing `--platform` for create and update commands), it will introduce a lot of complexity. Auto detecting platform information from config seems reasonable.

4. **Use positional arguments to specify the target and sources, instead of using options**

In the first iteration I used options to specify target index and its tag, making the list of manifests only command argument. Currently I use positional argument, so the first argument of the commands is the to-be-created/updated index with possible tags, and the remaining arguments are the list of manifests to add or remove. This aligns with many existing `oras` commands (`cp, attach, tag, manifest push`).

5. **Should we require all the source manifests and the to-be-created index to be in the same repository?**

Yes, as allowing multiple repositories will introduce a lot of copying of missing blobs and manifests.

6. **Should we automatically push the created/updated index? Or should we decouple creating/updating the index and pushing the index?**

I currently implement auto push, as storing the newly created index locally introduces a lot of complexities (e.g. more option flags, possible operations on the unpushed index). In the future we may introduce `--no-auto-push` option if necessary. 