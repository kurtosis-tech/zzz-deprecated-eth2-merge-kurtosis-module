# TBD
### Changes
* Suggest users store their module parameters in a file, so they're easier to work with

### Fixes
* Fix port IDs for Kurtosis 0.10.0

# 0.4.4
### Changes
* Upped the Lodestar wait-for-availability time to 60s
* Updated flags for Kiln testnet compatibility
* Switches key generation to use insecure mode, making key generation extremely fast

# 0.4.3
### Features
* Add extra debug logging to EL REST client, for debugging any issues
* Add new module's params `elLogLevel` and `clLogLevel` to configure a specific EL and CL client's log level
* Added the `elExtraParams`, `beaconExtraParams`, and `validatorExtraParams` properties to a participant to allow for overriding participant commands

### Changes
* Set config values to `BELLATRIX_` rather than `MERGE_`
* Set `ETH1_FOLLOW_DISTANCE` to 30

### Fixes
* Use emptystring for Besu ENR, as there's no way to get it right now without the logs
* Fixed a Teku break caused by a flag getting renamed in the latest version of the Teku image

# 0.4.2
### Features
* Added generation of Besu genesis file
* Added Besu EL

### Fixes
* Fixed an issue where the CL REST client would try to deserialize the bodies of responses that came back with non-200 status codes
* When a Teku node is present, require merge fork epoch to be >= 3 as a workaround for a bug in Teku
* Disallow a Prysm node being a boot node due to https://github.com/kurtosis-tech/eth2-merge-kurtosis-module/pull/36

### Changes
* Set the `mergeForkBlock` parameter in the EL genesis config template to `10` per Pari's recommendation
* Switch back to [the default genesis generator](https://github.com/skylenet/ethereum-genesis-generator) (rather than the Kurtosis fork of it)
* Nethermind genesis JSON is generated using the genesis generator image
* Centralized EL client availability-waiter and mining-waiter logic

# 0.4.1
### Fixes
* Fixed an issue where using emptystring as the default image wasn't working

# 0.4.0
### Features
* Allowed configurable EL & CL client images via the `elImage` and `clImage` keys to participant object
* Support `trace` loglevel

### Breaking Changes
* The participant spec's `el` and `cl` keys have been switched to `elType` and `clType`

# 0.3.0
### Features
* Added Prysm CL (beacon, validator) Launcher
* Made EL & CL client log levels configurable as a module param, `logLevel`
* Added self-documenting code for module params
* When an invalid EL or CL client type is provided in the params, the valid values are printed to the user
* Added a `waitForMining` property to the config, to allow users to skip the EL client mine-waiting (only useful for debugging a CL client)

### Changes
* The `WaitForBeaconClientAvailability` method also checks if the returned status is READY, which means the node is synced
* Replaced the custom implementation of the availability waiter method in Lodestar Launcher with the `WaitForBeaconClientAvailability` used for other launchers
* Set the Eth1 block time to 1 second in the CL config
* Revert back to the original Lodestar image, and comment out the `BELLATRIX_` config values for now

### Fixes
* Set the `--subscribe-all-subnets` flag equivalents on all Beacon nodes
* Generate the CL genesis files AFTER the EL network is mining, so that the CL network doesn't skip any important epochs (e.g. Altair, or merge fork) which causes it to get in a stuck state
* Removed unneeded hanging-around delay that existed in wait-for-finalization logic
* Wait until all CL nodes are up before starting to process slots
* Make forkmon respond to slots-per-epoch config changes
* Bump Lodestar wait-for-availability time up to 30s
* Don't launch Forkmon until CL genesis has been hit, due to a bug where if it receives a non-200 healthcheck status for a node then it won't ever revisit the node
* Updated the Geth image to `parithoshj/geth:merge-f72c361` (from around 2022-01-18)
* Updated TODOs in README

# 0.2.3
### Features
* Added the ability to specify arbitrary numbers of participants with EL/CL combos, and default to one Geth+Nimbus participant
* Added instructions to the README for configuring the module

### Fixes
* Get rid of the 300-second delay in the generated CL genesis
* Added the Lighthouse validator node

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
