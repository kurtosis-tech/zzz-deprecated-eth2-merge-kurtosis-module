My Kurtosis Module
=====================
Welcome to your new Kurtosis module! You can use the ExampleExecutableKurtosisModule implementation as a pattern to create your own Kurtosis module.

Quickstart steps:
1. Customize your own Kurtosis module by editing the generated files inside the `/path/to/your/code/repos/kurtosis-module/impl` folder
    1. Rename files and objects, if you want, using a name that describes the functionality of your Kurtosis module
    1. Write the functionality of your Kurtosis module inside your implementation of the `ExecutableKurtosisModule.execute` method by using the serialized parameters (validating & sanitizing the parameters as necessary)
    1. Write an implementation of `KurtosisModuleConfigurator` that accepts configuration parameters and produces an instance of your custom Kurtosis module
    1. Edit the main file and replace the example `KurtosisModuleConfigurator` with your own implementation that produces your custom `ExecutableKurtosisModule`
    1. Run `scripts/build.sh` to package your Kurtosis module into a Docker image that can be used inside Kurtosis
