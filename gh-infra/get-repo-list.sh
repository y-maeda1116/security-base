#!/bin/bash
gh repo list y-maeda1116 --visibility public -L 100 --json nameWithOwner --jq '.[].nameWithOwner' > gh-infra/targets.txt
