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
1. Create a file in your home directory `eth2-module-params.json` with the following contents:

   ```yaml
   logLevel: "info"
   ```

1. Execute the module, passing in the params from the file:
   ```bash
   kurtosis module exec --enclave-id eth2 kurtosistech/eth2-merge-kurtosis-module --execute-params "$(cat ~/eth2-module-params.json)"
   ```

Management
----------

Kurtosis will create a new enclave to house the services of the Ethereum network. [This page][using-the-cli] contains documentation for managing the created enclave & viewing detailed information about it.

Configuration
-------------

To configure the module behaviour, you can modify your `eth2-module-params.json` file. The full JSON schema that can be passed in is as follows with the defaults ([from here](https://github.com/kurtosis-tech/eth2-merge-kurtosis-module/blob/master/kurtosis-module/impl/module_io/default_params.go) provided (though note that the `//` comments are for explanation purposes and aren't valid JSON so need to be removed):

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
   ```
   source scripts/_constants.env && \
       kurtosis enclave rm -f eth2-local && \
       bash scripts/build.sh && \
       kurtosis module exec --enclave-id eth2-local "${IMAGE_ORG_AND_REPO}:$(bash scripts/get-docker-image-tag.sh)" --execute-params "{}"
   ```
   NOTE 1: You can change the value of the `--execute-params` flag to pass in extra configuration to the module per the "Configuration" section above!
   NOTE 2: The `--execute-params` flag accepts YAML and YAML is a superset of JSON, so you can pass in either.

Documentation
-------------
This repo is a Kurtosis module. To get general information on what a Kurtosis module is and how it works, visit [the modules documentation](https://docs.kurtosistech.com/modules.html).

The overview of this particular module's operation is as follows:

1. Parse user parameters
1. Launch a network of Ethereum participants
    1. Generate execution layer (EL) client config data
    1. Launch EL clients
    1. Wait for EL clients to start mining, such that all EL clients have a nonzero block number
    1. Generate consensus layer (CL) client config data
    1. Launch CL clients
1. Launch auxiliary services (Grafana, Forkmon, etc.)
1. Run Ethereum Merge verification logic

### Architecture Overview
The module has six main components, in accordance with the above operation:

1. [Execute Function][execute-function]
1. [Module I/O][module-io]
1. [Participant Network][participant-network]
1. **Auxiliary Services:** All the various other directories under the `kurtosis-module/impl` directory, including `forkmon`, `grafana`, `prometheus`, and `transaction-spammer`
1. **Static Files:** The `kurtosis-module/static_files` directory, which contains static files that will get bundled up with the module (e.g. genesis file templates)
1. **Merge Verification Logic:** The `kurtosis-module/impl/testnet_verifier` directory, which contains checks that the network has successfully passed the Merge

### [Execute Function][execute-function]
The execute function is the module's entrypoint/main function, where parameters are received from the user, lower-level calls are made, and a response is returned. Like all Kurtosis modules, this module receives serialized parameters and [the EnclaveContext object for manipulating the Kurtosis enclave][enclave-context], and returns a serialized response object.

### [Module I/O][module-io]
This particular module has many configuration options (see the "Configuration" section earlier in this README for the full list of values). These are passed in as a YAML-serialized string, and arrive to the module's execute function via the `serializedParams` variable. The process of setting defaults, overriding them with the user's desired options, and validating that the resulting config object is valid requires some space in the codebase. All this logic happens inside the `module_io` directory, so you'll want to visit this directory if you want to:

- View or change parameters that the module can receive
- Change the default values of module parameters
- View or change the validation logic that the module applies to configuration parameters
- View or change the properties that the module passes back to the user after execution is complete

### [Participant Network][participant-network]
The participant network is the beating Ethereum network heart at the center of the module. The participant network code is responsible for:

1. Generating EL client config data
1. Starting the EL clients
1. Waiting until the EL clients have started mining
1. Generating CL client config data
1. Starting the CL clients

We'll explain these in stages.

#### EL clients
All EL clients require both a genesis file and a JWT secret. The exact format of the genesis file differs per client, so we first leverage [a Docker image containing tools for generating this genesis data][ethereum-genesis-generator] to create the actual files that the EL clients-to-be will need. These files get stored in the Kurtosis enclave, ready for use when we start the EL clients.

Next, we plug the generated genesis data into EL client "launchers" to start a mining network of EL nodes. The launchers are really just implementations of [the `ELClientLauncher` interface](https://github.com/kurtosis-tech/eth2-merge-kurtosis-module/blob/master/kurtosis-module/impl/participant_network/el/el_client_launcher.go), with a `Launch` function that consumes EL genesis data and produces information about the running EL client node. Running EL node information is represented by [an `ELClientContext` struct](https://github.com/kurtosis-tech/eth2-merge-kurtosis-module/blob/master/kurtosis-module/impl/participant_network/el/el_client_context.go). Each EL client type (e.g. Besu, Erigon, Geth) has its own launcher because each EL client will require different environment variables and flags to be set when launching the client's container.

Once we have a network of EL nodes started, we block until they all have a block number of > 0 (to ensure that they are in fact working). After the nodes have started mining, we're ready to move on to adding the CL client network.

CL clients, like EL clients, also have genesis and config files that they need. We use [the same Docker image with tools for generating genesis data][ethereum-genesis-generator] to create the files that the CL-clients-to-be need, and the files get stored in the Kurtosis enclave in the same way as the EL client files.

We accomplish these steps using several components:

- Launchers
- Prelaunch data generators
- Waiters

A launcher is a 

A prelaunch data generator is a function used to generate data for an EL or CL node before it's launched. Such data includes EL & CL genesis config data, JWT token secrets, CL client validator keystores







<!------------------------ Only links below here -------------------------------->

[docker-installation]: https://docs.docker.com/get-docker/
[kurtosis-cli-installation]: https://docs.kurtosistech.com/installation.html
[module-docs]: https://docs.kurtosistech.com/modules.html
[enclave-context]: https://docs.kurtosistech.com/kurtosis-core/lib-documentation#enclavecontext
[using-the-cli]: https://docs.kurtosistech.com/using-the-cli.html

[execute-function]: https://github.com/kurtosis-tech/eth2-merge-kurtosis-module/blob/master/kurtosis-module/impl/module.go#L50
[module-io]: https://github.com/kurtosis-tech/eth2-merge-kurtosis-module/tree/master/kurtosis-module/impl/module_io
[participant-network]: https://github.com/kurtosis-tech/eth2-merge-kurtosis-module/tree/master/kurtosis-module/impl/participant_network
[ethereum-genesis-generator]: https://github.com/skylenet/ethereum-genesis-generator
