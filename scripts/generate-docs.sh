#!/usr/bin/env bash

# Copyright (c) Ryan Johnson
# SPDX-License-Identifier: MPL-2.0

componentTypeFromFolderName() {
    if [[ "$1" = "builders" ]]; then
        echo "builder"
    elif [[ "$1" = "provisioners" ]]; then
        echo "provisioner"
    elif [[ "$1" = "post-processors" ]]; then
        echo "post-processor"
    elif [[ "$1" = "datasources" ]]; then
        echo "data-source"
    else
        echo ""
    fi
}

rewriteLinks() {
  local result="$1"
  local organization="$2"

  urlSegment="([^/]+)"
  urlAnchor="(#[^/]+)"

  local find="\(\/packer\/plugins\/$urlSegment\/$urlSegment$urlAnchor?\)"
  local replace="\(\/packer\/integrations\/$organization\/\2\3\)"
  result="$(echo "$result" | sed -E "s/$find/$replace/g")"

  local find="\(\/packer\/plugins\/$urlSegment\/$urlSegment\/$urlSegment$urlAnchor?\)"
  local replace="\(\/packer\/integrations\/$organization\/\2\/latest\/components\/\1\/\3\4\)"
  result="$(echo "$result" | sed -E "s/$find/$replace/g")"

  result="$(echo "$result" \
      | sed "s/\/datasources\//\/data-source\//g" \
      | sed "s/\/builders\//\/builder\//g" \
      | sed "s/\/post-processors\//\/post-processor\//g" \
      | sed "s/\/provisioners\//\/provisioner\//g" \
  )"

  echo "$result"
}

processComponentFile() {
    local docsDir="$1"
    local webDocsDir="$2"
    local componentFile="$3"

    local escapedDocsDir="$(echo "$docsDir" | sed 's/\//\\\//g' | sed 's/\./\\\./g')"
    local componentTypeAndSlug="$(echo "$componentFile" | sed "s/$escapedDocsDir\///g" | sed 's/\.mdx//g')"

    local componentSlug="$(echo "$componentTypeAndSlug" | cut -d'/' -f 2)"
    local componentType="$(componentTypeFromFolderName "$(echo "$componentTypeAndSlug" | cut -d'/' -f 1)")"
    if [[ "$componentType" = "" ]]; then
        echo "Failed to process '$componentFile', unexpected folder name."
        echo "Documentation for components must be stored in one of:"
        echo "builders, provisioners, post-processors, datasources"
        exit 1
    fi

    local webDocsFolder="$webDocsDir/components/$componentType/$componentSlug"
    mkdir -p "$webDocsFolder"
    local webDocsFile="$webDocsFolder/README.md"
    local webDocsFileTmp="$webDocsFolder/README.md.tmp"

    cp "$componentFile" "$webDocsFile"

    local lastMetadataLine="$(grep -n -m 2 '^\-\-\-' "$componentFile" | tail -n1 | cut -d':' -f1)"
    cat "$webDocsFile" | tail -n +"$(($lastMetadataLine+2))"  > "$webDocsFileTmp"
    mv "$webDocsFileTmp" "$webDocsFile"

    cat "$webDocsFile" | tail -n +3 > "$webDocsFileTmp"
    mv "$webDocsFileTmp" "$webDocsFile"

    rewriteLinks "$(cat "$webDocsFile")" "$4" > "$webDocsFileTmp"
    mv "$webDocsFileTmp" "$webDocsFile"
}

compileWebDocs() {
  local docsDir="$1/$2"
  local webDocsDir="$1/$3"

  echo "Compiling MDX docs in '$2' to Markdown in '$3'..."
  mkdir -p "$webDocsDir"

  cp "$docsDir/README.md" "$webDocsDir/README.md"

  for file in $(find "$docsDir" | grep "$docsDir/.*/.*\.mdx" | grep --invert-match "index.mdx"); do
    processComponentFile "$docsDir" "$webDocsDir" "$file" "$4"
  done
}

compileWebDocs "$1" "$2" "$3" "$4"
