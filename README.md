Ethereum 2 Merge Module
=======================
This is a [Kurtosis module][module-docs] that will:

1. Spin up a network of mining Eth1 clients
1. Spin up a network of Eth2 Beacon/validator clients
1. Add [a transaction spammer](https://github.com/kurtosis-tech/tx-fuzz) that will repeatedly send transactions to the network
1. Launch [a consensus monitor](https://github.com/ralexstokes/ethereum_consensus_monitor) instance attached to the network
1. Perform the merge
1. Optionally block until the Beacon nodes finalize an epoch (i.e. finalized_epoch > 0 and finalized_epoch = current_epoch - 3)

### Quickstart
1. [Install Docker if you haven't done so already][docker-installation]
1. [Install the Kurtosis CLI, or upgrade it to the latest version if it's already installed][kurtosis-cli-installation]
1. Ensure your Docker engine is running:
    ```
    docker image ls
    ```
1. Execute the module:
    ```
    kurtosis module exec --enclave-id eth2 kurtosistech/eth2-merge-kurtosis-module --execute-params '{}'
    ```

To configure the module behaviour, provide a non-empty JSON object to the `--execute-params` flag. The JSON schema that can be passed in is defined HERETODOOOOO, and the default values that will be used if you omit fields are defined HERETODOOOOOO.

### Management
Kurtosis will create a new enclave to house the services of the Ethereum network. [This page][using-the-cli] contains documentation for managing the created enclave & viewing detailed information about it.

<!-- Only links below here -->
[docker-installation]: https://docs.docker.com/get-docker/
[kurtosis-cli-installation]: https://docs.kurtosistech.com/installation.html
[module-docs]: https://docs.kurtosistech.com/modules.html
[using-the-cli]: https://docs.kurtosistech.com/using-the-cli.html

### Development
The unit tests in this module also require Kurtosis to be available.
