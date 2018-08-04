#!/bin/sh
set -e
cd public
git config --global user.email "jiangkun.zhao90@gmail.com"
git config --global user.name "Zhao Jiangkun"
git init
git add .
git commit -m "Deploy to Github pages"
git checkout -b gh-pages
git push --force --quiet origin gh-pages
echo "Finished Deployment"
