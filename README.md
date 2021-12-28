Ethereum 2 Merge Module
=======================
This is a [Kurtosis module][module-docs] that does the following:

1. Spins up two merge-ready Geth EL clients
1. Spins up two Nimbus CL clients
1. Adds [a transaction spammer](https://github.com/kurtosis-tech/tx-fuzz) that will repeatedly send transactions to the network
1. Launches [a consensus monitor](https://github.com/ralexstokes/ethereum_consensus_monitor) instance attached to the network
1. Waits until epoch finalization occurs (i.e. finalized_epoch > 0 and finalized_epoch = current_epoch - 3)

### Quickstart
1. [Install Docker if you haven't done so already][docker-installation]
1. [Install the Kurtosis CLI, or upgrade it to the latest version if it's already installed][kurtosis-cli-installation]
1. Ensure your Docker engine is running:
    ```
    docker image ls
    ```
1. Execute the module:
    ```
    kurtosis module exec --enclave-id eth2 kurtosistech/eth2-merge-kurtosis-module
    ```

Kurtosis will create a new enclave to house the services of the Ethereum network. [This page][using-the-cli] contains documentation for managing the enclave & viewing detailed information about it.

<!-- Only links below here -->
[docker-installation]: https://docs.docker.com/get-docker/
[kurtosis-cli-installation]: https://docs.kurtosistech.com/installation.html
[module-docs]: https://docs.kurtosistech.com/modules.html
[using-the-cli]: https://docs.kurtosistech.com/using-the-cli.html
