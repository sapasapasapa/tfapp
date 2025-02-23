# tfapp

This initial version:
1. Checks if terraform is installed
2. Executes `terraform apply`
3. Forwards all output to the console
4. Allows for interactive confirmation (the default terraform prompt)

To build and test the application, you can:
1. Navigate to the project directory
2. Run `go build -o build/tfapp ./cmd/app`
3. Run the resulting binary