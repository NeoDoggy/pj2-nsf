# Network Service Function

This is a network function used for network service register/de-register/discovery.

## SBI Api Endpoint Point

### API Table

| Service | Endpoint | Description | Model |
| - | - | - | - |
| Heartbeat | `/sbi/heartbeat` | health check | - |
| NF Management | `/sbi/nnsf-nfm` | register & de-register network service | [nnsf-nfm](#nnsf-nfm) |
| Service Discovery | `/sbi/nnsf-disc` | discover the service information | [nnsf-disc](#nnsf-disc) |
| Group Service Discovery | `/sbi/nnsf-disc/group` | discover multiple services information | [nnsf-disc](#nnsf-disc) |

### API Structure

```bash
/sbi
    └─/heartbeat(GET)
    └─/nnsf-nfm(POST\DELETE)
    └─/nnsf-disc(Get)
        └─/group(Get)
```

## Make Command

| Command | Description |
| - | - |
| `make` | default set to `make bin` |
| `make bin` | build the executable binary file and output under **build** folder |
| `make clean` | clean up the build folder |
| `make lint` | run golang-lint |
| `make test` | run go test through the whole repo |
| `make tidy` | run module tidy |
| `make run` | run the NSF binary file under **build** folder with [config.yaml](./config.yaml) |
| `make cleandb` | remove the bbolt db file |
| `make cleantestdb` | remove the test bbolt db files |

## Service Context

Services are stored by service name, with the NF name, SBI address and port encoded as JSON. Discovery reads the database directly, so registered services remain available after restarting NSF. Deregistration checks the NF name before removing services. Each service name has one provider; registering the same name replaces its stored details.

## MongoDB Interface

The default `nsf.db_type` in `config.yaml` remains `bbolt` for grading. To demonstrate the custom DB:

1. Start a local MongoDB server at `mongodb://127.0.0.1:27017`.
2. Change `nsf.db_type` to `mongodb` in `config.yaml`.
3. Run `make` and `make run`, then register services through the SBI API.
4. Inspect the records using `mongosh`:

    ```bash
    mongosh 'mongodb://127.0.0.1:27017/nsf' --eval 'db.services.find().toArray()'
    ```

5. Stop NSF and restore `nsf.db_type` to `bbolt` before submitting.

MongoDB uses database `nsf`, collection `services`, with the service name in `_id` and the JSON string in `value`. Operations have a five-second timeout. Batch saves write one key at a time and are not transactional.

The default `make test` skips MongoDB tests. With local MongoDB running, explicitly enable them with:

```bash
NSF_TEST_MONGODB=1 go test -v ./internal/db
```

This opt-in test removes all records in `nsf.services`; use a disposable local database and stop NSF first.

## Docker Image Validate

- Pull the image

    ```bash
    docker pull alonza0314/nsf-tester:latest
    ```

- Run the image for testing with local directory

    ```bash
    docker run --rm -v <absolute path to your local directory>:/work alonza0314/nsf-tester:latest
    ```

## SBI Model

### nnsf-nfm

- POST Request

    ```go
    type NnsfNfmPostRequest struct {
        NfName     string   `json:"nf_name"`
        SbiAddress string   `json:"sbi_address"`
        SbiPort    int      `json:"sbi_port"`
        Services   []string `json:"services"`
    }
    ```

- POST Response

    ```go
    type NnsfNfmPostResponse struct {
        ServiceToken string `json:"service_token"`
        ServiceUuid  string `json:"service_uuid"`
    }
    ```

- DELETE Request

    ```go
    type NnsfNfmDeleteRequest struct {
        NfName   string   `json:"nf_name"`
        Services []string `json:"services"`
    }
    ```

### nnsf-disc

- GET Request

    ```go
    type NnsfDiscGetRequest struct {
        Service string `json:"service"`
    }

    type NnsfDiscGroupGetRequest struct {
        Services []string `json:"services"`
    }
    ```

- GET Response

    ```go
    type NnsfDiscGetResponse struct {
        Service    string `json:"service"`
        NfName     string `json:"nf_name"`
        SbiAddress string `json:"sbi_address"`
        SbiPort    int    `json:"sbi_port"`
    }

    type NnsfDiscGroupGetResponse struct {
        Services []NnsfDiscGetResponse `json:"services"`
    }
    ```
