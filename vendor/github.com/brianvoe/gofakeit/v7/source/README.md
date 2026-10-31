# Random Number Generators Collection

This repository contains a collection of random number generators (RNGs) implemented in Go, designed to cater to a wide range of applications, from cryptographic operations to testing environments. Each RNG in the collection offers distinct features and performance characteristics, making it suitable for various use cases, including those requiring cryptographic security.

## Generators

### Crypto

- **Description**: Utilizes Go's `crypto/rand` package to provide cryptographically secure random numbers, suitable for security-sensitive applications.
- **Usage**:
  ```go
  source := NewCryptoSource()
  number := source.Uint64()
  ```

### JSF (Jenkins Small Fast)

- **Description**: An implementation of the Jenkins Small Fast hash function for efficient pseudo-random number generation, balancing speed and randomness quality for general use.
- **Usage**:
  ```go
  source := NewJSFSource(seed)
  number := source.Uint64()
  ```

### SFC (Simple Fast Counter)

- **Description**: Based on the Simple Fast Counter algorithm, this source offers rapid number generation with satisfactory randomness properties, ideal for simulations and non-cryptographic applications.
- **Usage**:
  ```go
  source := NewSFCSource(seed)
  number := source.Uint64()
  ```

### Dumb

- **Description**: A deterministic generator designed primarily for testing, providing predictable output for scenarios where consistent results are more beneficial than high-quality randomness.
- **Usage**:
  ```go
  source := NewDumb(seed)
  number := source.Uint64()
  ```

## Installation

To use these RNGs in your Go project, import the package as follows:

```go
import "github.com/yourusername/randsource"
```

Replace `yourusername` with your GitHub username or organization name where the repository is hosted.

## Usage

After importing the package, initialize the desired RNG with or without a seed (as applicable) and use the `Uint64` method to generate random numbers. See the usage examples under each generator's description for more details.

## Benchmarks

Performance benchmarks for each RNG are provided to help you choose the right generator for your application. These benchmarks cover various aspects, including speed and randomness quality.

For detailed benchmark results, see the [Benchmarks](https://github.com/brianvoe/gofakeit/blob/master/source/BENCHMARKS.md) file.

## Contributing

We welcome contributions and suggestions! Please open an issue or submit a pull request with your improvements.
