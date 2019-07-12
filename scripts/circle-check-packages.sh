#!/bin/bash

function parse_git_hash() {
  git rev-parse --short HEAD 2> /dev/null | sed "s/\(.*\)/\1/"
}

function prapare_version_commit () {
  echo $'\nCommiting version changes. This commit will not be checked-in!'
  git config --global user.email "circleci@grafana.com"
  git config --global user.name "CirceCI"
  git commit -am "Version commit"
}

#Get current version from lerna.json
PACKAGE_VERSION=`grep '"version"' lerna.json | cut -d '"' -f 4`
# Get short current commit's has
GIT_BRANCH=$(parse_git_hash)

echo "Commit: ${GIT_BRANCH}"
echo "Current lerna.json version: ${PACKAGE_VERSION}"

# count packages that changed
count=`npx lerna changed --loglevel silent | awk '{c++} END {print c}'`

if [ -z $count ]; then
  echo "No changes in packages, skipping packages publishing"
else
  echo "Changes detected in ${count} packages"
  echo "Releasing packages under ${PACKAGE_VERSION}-${GIT_BRANCH}"
  npx lerna version ${PACKAGE_VERSION}-${GIT_BRANCH} --no-git-tag-version --no-push --force-publish -y
  echo $'\nGit status:'
  git status -s

  echo $'\nBuilding packages'
  yarn packages:build

  prapare_version_commit


  echo $'\nPublishing packages'
  yarn packages:publishNext
fi

