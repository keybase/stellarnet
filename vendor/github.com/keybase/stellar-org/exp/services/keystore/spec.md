## Keystore Spec

### Problem

We need a keystore service that supports non-custodial applications.
It will make the process of stellarizing any applications easier as
they don't have to implement the logic to create a stellar account
and handle the encrypted private key themselves.

It is also intended to be the service that wallet SDK talks to.

### Authentication

For simplicity we will have each application spin up their own keystore
server, so there won’t be any routing logic in the keystore server that
directs requests to the correct client server to authenticate. Since we
don’t anticipate a lot of requests to the keystore from each user, we
should be able to tolerate having another round trip for relaying the
auth token to the client server.

Clients will have to configure a API endpoint on their servers used for
authentication when booting up the keystore. Please refer to [this section](#required-changes-in-client-server)
for more details.

An auth token needs to be passed with all requests, and that's why there
is no need for a userid argument to the endpoints below.

<img src=attachments/2019-04-24-keystore-auth-flows.png>

### Raw Key Data

*RawKeyData Object:*

```typescript
interface RawKeyData {
	keyType: string;
	publicKey: string;
	privateKey: string;
	path?: string;
	extra?: any;
}
```

```typescript
type RawKeys = RawKeyData[]
```

The clients will encrypt RawKeys with a salt based on the encrypter they use.
The clients will transmit the encrypted keys blob as a base64 URL encoded string.

### Encrypted Key Data

*EncryptedKeysData Object:*

```typescript
interface EncryptedKeysData {
	encrypterName: string;
	salt: string;
	keysBlob: string;
	creationTime: number;
	modifiedTime: number;	
}
```

We support three different kinds of HTTP methods to manipulate keys:

### PUT /keys

Put Keys Request:

```typescript
interface PutKeysRequest {
	encrypterName: string;
	salt: string;
	keysBlob: string;
}
```

Put Keys Response:

```typescript
type PutKeysResponse = EncryptedKeysData;
```

<details><summary>Errors</summary>

TBD
```json
{
	"code": "some error code",
	"message": "some error message",
	"retriable": false,
}
```
</details>

### GET /keys

Get Keys Request:

This endpoint will return the keys blob corresponding to the auth token
in the request header, if the token is valid. This endpoint does not take
any parameter.

Get Keys Response:

```typescript
type GetKeysResponse = EncryptedKeysData;
```
<details><summary>Errors</summary>

TBD
```json
{
	"code": "some error code",
	"message": "some error message",
	"retriable": false,
}
```
</details>

### DELETE /keys

Delete Keys Request:

This endpoint will delete the keys blob corresponding to the auth token
in the request header and return the deleted keys blob to the client, if
the token is valid. This endpoint does not take any parameter.

Delete Keys Response:

*Success:*

```typescript
interface Success {
	message: "ok";
}
```

<details><summary>Errors</summary>

TBD
```json
{
	"code": "some error code",
	"message": "some error message",
	"retriable": false,
}
```
</details>

### Required Changes in Client Server

Applications using the keystore will have to implement an endpoint
that takes an auth token and returns a userid in the following json format:

```json
{
	"userID": "some-user-id"
}
```

The keystore will send a HTTP GET request to your designated endpoint and
parse the result in the above format.
