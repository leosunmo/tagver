# Tagver

Simple Go binary that allows you to get latest Git SHAs/Tags (annotated or lightweight) and current Branch.

## Usage
```
Usage of tagver: [-t] [-b] [-c] [<git dir>]

Default output is very close to "git describe --tags":
	If HEAD is not tagged: <tag>-<commits since tag>-<HEAD SHA> (example: v1.0.4-1-5227b593)
	If HEAD is tagged: <tag> (example: v1.0.5)

If "-b" or "-c" are provided with "-t", only the tag name will print regardless if it's clean or not.
Print order will be <tag>-<branch>-<SHA>.

Set one or more flags.
  -b	Return the current branch
  -c	Return the current commit
  -ignore-unclean-tag
    	Return only tag name even if the latest tag doesn't point to HEAD ("v1.0.4" instead of "v1.0.4-1-89c22b28")
  -t	Return the latest semver tag (annotated or lightweight Git tag) (default)
```

## Output
Default with tag pointing to HEAD:
```
tagver
v1.0.4
```
Default with commits after latest tag:
```
tagver
v1.0.4-1-5227b593
```
Default with no tags present:
```
tagver
main-63380731
```
Default ignoring the fact that there's commits after the latest tag:
```
tagver --ignore-unclean-tag
v1.0.4
```
All options provided (ignores the fact that latest tag doesn't point to HEAD):
```
tagver -t -b -c
v1.0.4-main-5227b593
```