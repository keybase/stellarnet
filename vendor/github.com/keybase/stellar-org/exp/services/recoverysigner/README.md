# Recovery Signer

This is an incomplete and work-in-progress implementation of the [SEP-30]
Recovery Signer protocol v0.4.0.

A Recovery Signer is a server that can help a user regain control of a Stellar
account if they have lost their secret key. A user registers their account with
a Recovery Signer by adding it as a signer, and informs the Recovery Signer
that any user proving access to a phone number or email address can have
transactions signed. A user who has registered their account with two or more
Recovery Signers can recover the account with their help.

This implementation uses Firebase to authenticate a user with an email address
or phone number. To configure a Firebase project for use with recoverysigner
see [README-Firebase.md](README-Firebase.md).

This implementation is not polished and is still experimental.
Running this implementation in production is not recommended.

## Usage

```
$ recoverysigner --help
SEP-30 Recovery Signer server

Usage:
  recoverysigner [command] [flags]
  recoverysigner [command]

Available Commands:
  db          Run database operations
  serve       Run the SEP-30 Recovery Signer server

Use "recoverysigner [command] --help" for more information about a command.
```

## Usage: Serve

```
$ recoverysigner serve --help
Run the SEP-30 Recover Signer server

Usage:
  recoverysigner serve [flags]

Flags:
      --admin-port int               Port to listen and serve admin functionality including metrics (ADMIN_PORT)
      --db-url string                Database URL (DB_URL) (default "postgres://localhost:5432/?sslmode=disable")
      --firebase-project-id string   Firebase project ID to use for validating Firebase JWTs (FIREBASE_PROJECT_ID)
      --metrics-namespace string     Namespace to use for metric names prefixed to metrics reported (METRICS_NAMESPACE) (default "recoverysigner")
      --network-passphrase string    Network passphrase of the Stellar network transactions should be signed for (NETWORK_PASSPHRASE) (default "Test SDF Network ; September 2015")
      --port int                     Port to listen and serve on (PORT) (default 8000)
      --sep10-jwks string            JSON Web Key Set (JWKS) containing one or more keys used to validate SEP-10 JWTs (if the key is an asymmetric key that has separate public and private key, the JWK need only contain the public key) (if multiple keys are provided they will all attempt verification the key ID will be ignored although logged) (SEP10_JWKS)
      --sep10-jwt-issuer string      JWT issuer to verify if in the SEP-10 JWT iss field (not checked if empty) (SEP10_JWT_ISSUER)
      --signing-key string           Stellar signing key(s) used for signing transactions comma separated (first key is preferred signer) (will be deprecated with per-account keys in the future) (SIGNING_KEY)
```

[SEP-30]: https://github.com/stellar/stellar-protocol/blob/600c326b210d71ee031d7f3a40ca88191b4cdf9c/ecosystem/sep-0030.md
[README-Firebase.md]: README-Firebase.md
