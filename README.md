# Tagver

Simple Go binary that allows you to get latest Git SHAs/Tags (annotated or lightweight) and current Branch.

## Usage
```
Usage of tagver: [-t] [-b] [-c] [<git dir>]

Default output is very close to "git describe --tags" if there's a tag present, otherwise defaults to branch and commit:
	If HEAD is not tagged: <branch>-<HEAD SHA> (example: main-63380731)
	If HEAD is tagged: <tag>-<HEAD SHA> (example: v1.0.4-63380731)
	If HEAD is tagged but has commits after latest tag: <tag>-<commits since tag>-<HEAD SHA> (example: v1.0.4-1-5227b593)

If "-b" or "-c" are provided with "-t", it is additive and will include commits since tag if unclean.
The number of commits since tag can be ignored with "-ignore-unclean-tag".

Print order will be <tag>-<branch>[-<commits since tag>]-<SHA>.

Set one or more flags.
  -b	Return the current branch
  -c	Return the current commit
  -ignore-unclean-tag
    	Return only tag name even if the latest tag doesn't point to HEAD ("v1.0.4" instead of "v1.0.4-1-5227b593")
  -t	Return the latest semver tag (annotated or lightweight Git tag)
```

## Output
Default with tag pointing to HEAD:
```
tagver
v1.0.4-63380731
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
v1.0.4-5227b593
```
Only display tag, and ignore any commits after the latest tag:
```
tagver -t --ignore-unclean-tag
v1.0.4
```

All options provided (ignores the fact that latest tag doesn't point to HEAD):
```
tagver -t -b -c
v1.0.4-main-5227b593
```
## Verify signatures and checksums
The releases are signed with [cosign](https://cosign.io/). You can verify the signatures and checksums with the following commands:
```sh
# Verify the checksums.txt file
cosign verify-blob checksums.txt --cert checksums.txt.pem --signature checksums.txt.sig \
--certificate-oidc-issuer https://token.actions.githubusercontent.com \
--certificate-identity-regexp 'https://github\.com/leosunmo/tagver/\.github/workflows/build\.yml@refs/tags/v[0-9]+(\.[0-9]+){2}'

ARCH=x86_64
OS=Linux

# Verify the tagver binary
cosign verify-blob tagver_${OS}_${ARCH}.tar.gz --cert tagver_${OS}_${ARCH}.tar.gz.pem --signature tagver_${OS}_${ARCH}.tar.gz.sig \
--certificate-oidc-issuer https://token.actions.githubusercontent.com \
--certificate-identity-regexp 'https://github\.com/leosunmo/tagver/\.github/workflows/build\.yml@refs/tags/v[0-9]+(\.[0-9]+){2}'

# Verify the checksums
sha256sum --ignore-missing -c checksums.txt
```