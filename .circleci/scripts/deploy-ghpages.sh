#!/bin/sh
set -e
cd public
git init
git add .
git commit -m "Deploy to Github pages"
git checkout -b gh-pages
git push --force --quiet origin gh-pages
echo "Finished Deployment"
