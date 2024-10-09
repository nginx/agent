import os
import sys
import re
import requests

# GitHub environment variables
pr_number = os.getenv("PR_NUMBER")
repo_name = os.getenv("GITHUB_REPOSITORY")
token = os.getenv("GITHUB_TOKEN")

# Make sure we are using the latest PR data
url = f"https://api.github.com/repos/{repo_name}/pulls/{pr_number}"
headers = {"Authorization": f"token {token}"}
response = requests.get(url, headers=headers)

if response.status_code != 200:
    print("Error fetching PR details")
    sys.exit(1)

# Get the PR body
pr_body = response.json().get("body", "")

# Print the PR body for debugging (optional)
print("PR Body:", pr_body)

# Check for 'Proposed changes' section
proposed_changes_match = re.search(r"### Proposed changes\s+(.+)", pr_body, re.DOTALL)
if proposed_changes_match:
    proposed_changes_text = proposed_changes_match.group(1).strip()
    word_count = len(proposed_changes_text.split())
    
    if word_count <= 10:
        print(f"Error: 'Proposed changes' section should have more than 10 words. Found {word_count} words.")
        sys.exit(1)
else:
    print("Error: 'Proposed changes' section is missing.")
    sys.exit(1)

# Check if the first two checklist items are selected
# This pattern allows for optional spaces and both lowercase/uppercase x between brackets
contrib_checked = re.search(r"- \[\s*[xX]\s*\] I have read the \[`CONTRIBUTING`\]\(https://github.com/nginx/agent/blob/main/docs/CONTRIBUTING\.md\)", pr_body)
install_checked = re.search(r"- \[\s*[xX]\s*\] I have run `make install-tools`", pr_body)

if not contrib_checked:
    print("Error: The first checklist item is not checked.")
    sys.exit(1)

if not install_checked:
    print("Error: The second checklist item is not checked.")
    sys.exit(1)

print("PR description is valid.")