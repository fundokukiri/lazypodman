#!/usr/bin/env bash
# Fetch the Superpowers and Ponytail agent packages into this checkout so that
# free-pi / pi can use them, without adding anything to git.
#
# The repos are cloned into .pi/vendor/ (also registered as Pi packages in
# .pi/settings.json) and their skills are copied into the project skill
# directories .pi/skills/ and .agents/skills/, which the agent scans at startup.
# Those directories are gitignored: nothing here gets committed or pushed.
#
# Usage:
#   scripts/install-agent-packages.sh          # clone/update the packages and sync skills
#   scripts/install-agent-packages.sh --remove # delete the local install again

set -euo pipefail

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
vendor="$root/.pi/vendor"
settings="$root/.pi/settings.json"
skill_dirs=("$root/.pi/skills" "$root/.agents/skills")

packages=(
  "superpowers|https://github.com/obra/superpowers.git"
  "ponytail|https://github.com/DietrichGebert/ponytail.git"
)

if [[ "${1:-}" == "--remove" ]]; then
  rm -rf "$vendor" "$settings"
  for dir in "${skill_dirs[@]}"; do
    rm -rf "$dir"
  done
  echo "removed .pi/vendor, .pi/settings.json, .pi/skills and .agents/skills"
  exit 0
fi

command -v git >/dev/null || { echo "git is required" >&2; exit 1; }

mkdir -p "$vendor"

for entry in "${packages[@]}"; do
  name="${entry%%|*}"
  url="${entry#*|}"
  dir="$vendor/$name"

  if [[ -d "$dir/.git" ]]; then
    echo "updating $name"
    git -c safe.directory="$dir" -C "$dir" pull --ff-only --quiet
  else
    echo "cloning $name"
    git clone --depth 1 --quiet "$url" "$dir"
  fi
done

# Copy every skill directory (one that contains a SKILL.md) into the project
# skill directories, so both pi and free-pi discover them without package
# support.
for dir in "${skill_dirs[@]}"; do
  mkdir -p "$dir"
done

count=0
for entry in "${packages[@]}"; do
  name="${entry%%|*}"
  shopt -s nullglob
  for skill in "$vendor/$name/skills"/*/; do
    [[ -f "$skill/SKILL.md" ]] || continue
    skill_name="$(basename "$skill")"
    for dir in "${skill_dirs[@]}"; do
      rm -rf "${dir:?}/$skill_name"
      cp -R "$skill" "$dir/$skill_name"
    done
    count=$((count + 1))
  done
  shopt -u nullglob
done

if [[ -f "$settings" ]]; then
  echo "kept existing $settings; make sure it declares the two vendor packages"
else
  # Extensions come from the packages, skills come from the copies above: "/
  # filtering the package skills avoids loading every skill twice.
  cat >"$settings" <<'JSON'
{
  "packages": [
    { "source": "vendor/superpowers", "skills": [] },
    { "source": "vendor/ponytail", "skills": [] }
  ]
}
JSON
  echo "wrote $settings"
fi

echo
echo "synced $count skills into .pi/skills and .agents/skills"
echo "start free-pi in this directory and accept the project trust prompt;"
echo "the skills then appear as /skill:brainstorming, /skill:ponytail, ..."
echo "remove again with: scripts/install-agent-packages.sh --remove"
