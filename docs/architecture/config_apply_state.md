# Config Apply States 
```mermaid

flowchart TB
 subgraph subGraph0["Legend"]
        12["Internal Agent Status"]
        13["Data Plane Response Status"]
  end
    0["COMMAND_STATUS_IN_PROGRESS"] --> 1["COMMAND_STATUS_IN_OK"]
    2["COMMAND_STATUS_IN_PROGRESS"] --> 3["COMMAND_STATUS_IN_FAILURE"]
    4["COMMAND_STATUS_IN_PROGRESS"] --> 5["COMMAND_STATUS_ERROR"]
    5 --> 6["COMMAND_STATUS_IN_PROGRESS"]
    6 --> 7["COMMAND_STATUS_ERROR"]
    8["COMMAND_STATUS_IN_PROGRESS"] --> 9["COMMAND_STATUS_ERROR"]
    9 --> 10["COMMAND_STATUS_IN_PROGRESS"]
    10 --> 11["COMMAND_STATUS_ERROR"]
    style 0 fill:#FFF9C4
    style 1 fill:#BBDEFB
    style 2 fill:#FFF9C4
    style 3 fill:#BBDEFB
    style 4 fill:#FFF9C4
    style 5 fill:#BBDEFB
    style 6 fill:#FFF9C4
    style 7 fill:#BBDEFB
    style 8 fill:#FFF9C4
    style 9 fill:#BBDEFB
    style 10 fill:#FFF9C4
    style 11 fill:#BBDEFB
    style 12 fill:#FFF9C4
    style 13 fill:#BBDEFB
    style subGraph0 stroke:#000000


```
