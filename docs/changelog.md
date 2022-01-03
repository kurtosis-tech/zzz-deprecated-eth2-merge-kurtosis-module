# TBD
### Fixes
* Get rid of the 300-second delay in the generated CL genesis

# 0.2.2
### Features
* Added a functioning Nimbus CL client
* Added Nethermind EL client
* Added a Lodestar CL (beacon, validator) Launcher
* * Added new `GetNodeSyncingData` method in cl rest client
* Added a transaction spammer to blast the network with transactions after all the nodes come up
* Added optional waiting until epoch finalization occurs

# 0.2.1
* Empty commit to try and kick CircleCI into actually building the tag

# 0.2.0
### Features
* Add build infra
* The Geth + Lighthouse node inside of Kurtosis now syncs with merge-devnet-3!
* Successfully-working private, mining network!
* Added a network of consensus-layer clients
* Hooked up genesis generation for Geth & CL nodes
* Lighthouse nodes peer with each other
* Add forkmon to the started network
* Add Teku CL client
* Enable CI

### Fixes
* Correct merge parameters like TTD, Altair fork version, merge fork version, etc. per Parithosh's recommendations
* Give Teku nodes 120s to start

### Changes
* Refactor the structure to reflect that there should be one EL node per CL node (and prepare for separated Beacon/validator nodes, like Lighthouse does)

# 0.1.0
* Initial commit
