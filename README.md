# NGINX Agent benchmark-results Branch

This branch is used to store the results of the performance tests ran on the `v3` pipeline. 
These results are then used to compare the performance test results of PRs against the results from v3 to check for decreasing performance.

### Files
`data.js` contains all the data from past runs of the performance tests on the v3 branch, this is used in the pipeline to check for decreasing performance.

`index.html` creates graphs from `data.js` to visualise the changes in performance over time. 


