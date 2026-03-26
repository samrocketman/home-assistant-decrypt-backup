#!/usr/bin/env bash
set -euo pipefail

# Upgrade all pinned GitHub Actions in .github/workflows/*.yml to the
# latest commit SHA for their version tag.
#
# Expected line format:
#   - uses: owner/repo@<sha> # vN
#
# Requires: curl, python3 (for JSON parsing)

WORKFLOW_DIR="${1:-.github/workflows}"

resolve_tag_sha() {
  local repo="$1" tag="$2"
  local ref sha type

  ref=$(curl -sf "https://api.github.com/repos/${repo}/git/ref/tags/${tag}") || return 1
  sha=$(echo "$ref" | python3 -c "import sys,json; print(json.load(sys.stdin)['object']['sha'])")
  type=$(echo "$ref" | python3 -c "import sys,json; print(json.load(sys.stdin)['object']['type'])")

  if [ "$type" = "tag" ]; then
    sha=$(curl -sf "https://api.github.com/repos/${repo}/git/tags/${sha}" \
      | python3 -c "import sys,json; print(json.load(sys.stdin)['object']['sha'])")
  fi
  echo "$sha"
}

declare -A seen
updated=0
skipped=0

for file in "${WORKFLOW_DIR}"/*.yml; do
  [ -f "$file" ] || continue

  while IFS= read -r match; do
    action=$(echo "$match" | sed 's/.*uses: *//; s/@.*//')
    old_sha=$(echo "$match" | sed 's/.*@//; s/ .*//')
    tag=$(echo "$match" | sed 's/.*# *//; s/ *$//')

    key="${action}@${tag}"
    if [ -n "${seen[$key]:-}" ]; then
      new_sha="${seen[$key]}"
    else
      new_sha=$(resolve_tag_sha "$action" "$tag" 2>/dev/null) || {
        echo "SKIP  ${action} ${tag} (API error)" >&2
        skipped=$((skipped + 1))
        continue
      }
      seen[$key]="$new_sha"
    fi

    if [ "$old_sha" = "$new_sha" ]; then
      echo "OK    ${action} ${tag} already up to date"
    else
      sed -i "s|${action}@${old_sha}|${action}@${new_sha}|g" "$file"
      echo "UP    ${action} ${tag} ${old_sha:0:12} -> ${new_sha:0:12} (${file})"
      updated=$((updated + 1))
    fi
  done < <(grep -h 'uses:.*@.*#' "$file" || true)
done

echo ""
echo "Done. ${updated} updated, ${skipped} skipped."
