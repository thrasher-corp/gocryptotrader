This folder contains optional git hooks to use when commiting to GoCryptoTrader

# To use:
## Command method:
Run the following:
`git config core.hooksPath .githooks`
This will set the hook path to this folder

# Git Hooks:
The following table details our git hooks and what they do

<table>
<thead>
<tr>
<th>Git Hook</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>Pre-Commit</td>
<td>Scans all test files and verifies that the testAPIKey, testAPISecret, clientID and canManipulateRealOrders fields will not comprimise you/the GoCryptoTrader repository. It will also perform the same checks against testconfig.json</td>
</tr></tbody></table>
