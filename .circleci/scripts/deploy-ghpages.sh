#!/bin/sh
set -e
changefile="changed.data"
if [ -f "$changefile" ]
then
    echo "$file found."
else
    echo "$file not found"
    echo "posts no changed"
    exit 0
fi
cp -R static/* public
cd public
git config --global user.email ${RELEASE_EMAIL}
git config --global user.name ${RELEASE_USERNAME}
git init
git remote add origin ${CIRCLE_REPOSITORY_URL}
git add .
git commit -m "Deploy to Github pages"
git checkout -b ${RELEASE_BRANCH}
git push --force --quiet origin ${RELEASE_BRANCH}
echo "Finished Deployment"
