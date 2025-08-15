# Testing Graceful Shutdown (SIGTERM)

To properly simulate a production environment (like Docker or Kubernetes) and test the `SIGTERM` signal handling, you should not use `go run`. The `go run` command creates a parent process that may not correctly forward signals to the compiled application (the child process).

The correct way is to first compile the application into a standalone binary and then run it directly. This ensures that when you send a signal, it is received directly by your application's process.

## Steps to Test SIGTERM Handling

### 1. Build the Executable

In your terminal, run the `go build` command to create a binary. We'll name the output `todoapi`.

```bash
go build -o todoapi .
```

This will create an executable file named todoapi in your project's root directory.

### 2. Run the Binary

Execute this binary directly instead of using go run:

```bash
./todoapi
```

The server will start up as usual. Now, it is running as a single, self-contained process, which accurately reflects how it would run in a production environment.

### 3. Send the `SIGTERM` Signal

Open a second terminal window to perform the test.

First, find the Process ID (PID) of your running application:

```bash
pgrep todoapi
```

Let's assume the command returns a PID of 54321.

Next, send the SIGTERM signal to that PID using the kill command. This is exactly what systems like Kubernetes do when they begin terminating a pod.

```bash
# Explicitly send the SIGTERM signal
kill -TERM 54321
```

Alternatively, you can use the kill command without specifying the signal, as SIGTERM is the default:

```bash
kill 54321
```

After sending the signal, switch back to your first terminal (where the server is running). You should see the graceful shutdown logs being printed, confirming that your application handled the `SIGTERM` signal correctly:

```bash
Shutting down gracefully, press Ctrl+C again to force
Server exiting
```

This method provides an accurate way to test your application's graceful shutdown logic, ensuring it will behave correctly when deployed and managed by orchestrators like Kubernetes.
