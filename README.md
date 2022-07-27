Ethereum 2 Merge Module
=======================

This is a [Kurtosis module][module-docs] that will:

1. Generate EL & CL genesis information using [this genesis generator](https://github.com/skylenet/ethereum-genesis-generator)
1. Spin up a network of mining Eth1 clients
1. Spin up a network of Eth2 Beacon/validator clients
1. Add [a transaction spammer](https://github.com/kurtosis-tech/tx-fuzz) that will repeatedly send transactions to the network
1. Launch [a consensus monitor](https://github.com/ralexstokes/ethereum_consensus_monitor) instance attached to the network
1. Perform the merge
1. Optionally block until the Beacon nodes finalize an epoch (i.e. finalized_epoch > 0 and finalized_epoch = current_epoch - 3)

For much more detailed information about how the merge works in Ethereum testnets, see [this document](https://notes.ethereum.org/@ExXcnR0-SJGthjz1dwkA1A/H1MSKgm3F).

Quickstart
----------

1. [Install Docker if you haven't done so already][docker-installation]
1. [Install the Kurtosis CLI, or upgrade it to the latest version if it's already installed][kurtosis-cli-installation]
1. Ensure your Docker engine is running:
   ```bash
   docker image ls
   ```
1. Create a file in your home directory `eth2-module-params.yaml` with the following contents:

   ```yaml
   logLevel: "info"
   ```

1. Execute the module, passing in the params from the file:
   ```bash
   kurtosis module exec --enclave-id eth2 kurtosistech/eth2-merge-kurtosis-module --execute-params "$(cat ~/eth2-module-params.yaml)"
   ```

Management
----------

Kurtosis will create a new enclave to house the services of the Ethereum network. [This page][using-the-cli] contains documentation for managing the created enclave & viewing detailed information about it.

Configuration
-------------

To configure the module behaviour, you can modify your `eth2-module-params.yaml` file. The full YAML schema that can be passed in is as follows with the defaults ([from here](https://github.com/kurtosis-tech/eth2-merge-kurtosis-module/blob/master/kurtosis-module/impl/module_io/default_params.go) provided:

You can find the latest Kiln compatible docker images here: https://notes.ethereum.org/@launchpad/kiln

```yaml
#  Specification of the participants in the network
participants:
  #  The type of EL client that should be started
  #  Valid values are "geth", "nethermind", and "besu"
  - elType: "geth"

    #  The Docker image that should be used for the EL client; leave blank to use the default for the client type
    #  Defaults by client:
    #  - geth: ethereum/client-go:latest
    #  - erigon: thorax/erigon:devel
    #  - nethermind: nethermind/nethermind:latest
    #  - besu: hyperledger/besu:develop
    elImage: ""

    #  The log level string that this participant's EL client should log at
    #  If this is emptystring then the global `logLevel` parameter's value will be translated into a string appropriate for the client (e.g. if
    #   global `logLevel` = `info` then Geth would receive `3`, Besu would receive `INFO`, etc.)
    #  If this is not emptystring, then this value will override the global `logLevel` setting to allow for fine-grained control
    #   over a specific participant's logging
    elLogLevel: ""

    #  A list of optional extra params that will be passed to the EL client container for modifying its behaviour
    elExtraParams: []

    #  The type of CL client that should be started
    #  Valid values are "nimbus", "lighthouse", "lodestar", "teku", and "prysm"
    clType: "lighthouse"

    #  The Docker image that should be used for the EL client; leave blank to use the default for the client type
    #  Defaults by client (note that Prysm is different in that it requires two images - a Beacon and a validator - separated by a comma):
    #  - lighthouse: sigp/lighthouse:latest
    #  - teku: consensys/teku:latest
    #  - nimbus: parithoshj/nimbus:merge-d3a00f6
    #  - prysm: gcr.io/prysmaticlabs/prysm/beacon-chain:latest,gcr.io/prysmaticlabs/prysm/validator:latest
    #  - lodestar: chainsafe/lodestar:next
    clImage: ""

    #  The log level string that this participant's EL client should log at
    #  If this is emptystring then the global `logLevel` parameter's value will be translated into a string appropriate for the client (e.g. if
    #   global `logLevel` = `info` then Teku would receive `INFO`, Prysm would receive `info`, etc.)
    #  If this is not emptystring, then this value will override the global `logLevel` setting to allow for fine-grained control
    #   over a specific participant's logging
    clLogLevel: ""

    #  A list of optional extra params that will be passed to the CL client Beacon container for modifying its behaviour
    #  If the client combines the Beacon & validator nodes (e.g. Teku, Nimbus), then this list will be passed to the combined Beacon-validator node
    beaconExtraParams: []

    #  A list of optional extra params that will be passed to the CL client validator container for modifying its behaviour
    #  If the client combines the Beacon & validator nodes (e.g. Teku, Nimbus), then this list will also be passed to the combined Beacon-validator node
    validatorExtraParams: []

    # TODO Probably want a mev-boost toggle, so that mev-boost only gets added if you flip it on

#  Configuration parameters for the Eth network
network:
  #  The network ID of the Eth1 network
  networkId: "3151908"

  #  The address of the staking contract address on the Eth1 chain
  depositContractAddress: "0x4242424242424242424242424242424242424242"

  #  Number of seconds per slot on the Beacon chain
  secondsPerSlot: 12

  #  Number of slots in an epoch on the Beacon chain
  slotsPerEpoch: 32

  #  Must come before the merge fork epoch
  #  See https://notes.ethereum.org/@ExXcnR0-SJGthjz1dwkA1A/H1MSKgm3F
  altairForkEpoch: 1

  #  Must occur before the total terminal difficulty is hit on the Eth1 chain
  #  See https://notes.ethereum.org/@ExXcnR0-SJGthjz1dwkA1A/H1MSKgm3F
  mergeForkEpoch: 2

  #  Once the total difficulty of all mined blocks crosses this threshold, the Eth1 chain will
  #   merge with the Beacon chain
  #  Must happen after the merge fork epoch on the Beacon chain
  #  See https://notes.ethereum.org/@ExXcnR0-SJGthjz1dwkA1A/H1MSKgm3F
  totalTerminalDifficulty: 100000000

  #  The number of validator keys that each CL validator node should get
  numValidatorKeysPerNode: 64

  #  This mnemonic will a) be used to create keystores for all the types of validators that we have and b) be used to generate a CL genesis.ssz that has the children
  #   validator keys already preregistered as validators
  preregisteredValidatorKeysMnemonic: "giant issue aisle success illegal bike spike question tent bar rely arctic volcano long crawl hungry vocal artwork sniff fantasy very lucky have athlete"

#  If set to false, we won't wait for the EL clients to mine at least 1 block before proceeding with adding the CL clients
#  This is purely for debug purposes; waiting for blockNumber > 0 is required for the CL network to behave as
#   expected, but that wait can be several minutes. Skipping the wait can be a good way to shorten the debug loop on a
#   CL client that's failing to start.
waitForMining: true

#  If set, the module will block until a finalized epoch has occurred.
#  If `waitForVerifications` is set to true, this extra wait will be skipped.
waitForFinalization: false

#  If set to true, the module will block until all verifications have passed
waitForVerifications: false

#  If set, this will be the maximum number of epochs to wait for the TTD to be reached.
#  Verifications will be marked as failed if the TTD takes longer.
verificationsTTDEpochLimit: 5

#  If set, after the merge, this will be the maximum number of epochs wait for the verifications to succeed.
verificationsEpochLimit: 5

#  The global log level that all clients should log at
#  Valid values are "error", "warn", "info", "debug", and "trace"
#  This value will be overridden by participant-specific values
logLevel: "info"
```

Development
-----------
First, install prerequisites:
1. Install Go
1. [Install Kurtosis itself](https://docs.kurtosistech.com/installation.html)

Then, run the dev loop:
1. Make your code changes
1. Rebuild and re-execute the module by running the following from the root of the repo:
   ```bash
   source scripts/_constants.env && \
       kurtosis enclave rm -f eth2-local && \
       bash scripts/build.sh && \
       kurtosis module exec --enclave-id eth2-local "${IMAGE_ORG_AND_REPO}:$(bash scripts/get-docker-image-tag.sh)" --execute-params "{}"
   ```
   NOTE 1: You can change the value of the `--execute-params` flag to pass in extra configuration to the module per the "Configuration" section above!
   NOTE 2: The `--execute-params` flag accepts YAML and YAML is a superset of JSON, so you can pass in either.

To get detailed information about the structure of the module, visit [the architecture docs](./docs/architecture.md).

<!------------------------ Only links below here -------------------------------->
[docker-installation]: https://docs.docker.com/get-docker/
[kurtosis-cli-installation]: https://docs.kurtosistech.com/installation.html
[module-docs]: https://docs.kurtosistech.com/modules.html
[enclave-context]: https://docs.kurtosistech.com/kurtosis-core/lib-documentation#enclavecontext
[using-the-cli]: https://docs.kurtosistech.com/using-the-cli.html
