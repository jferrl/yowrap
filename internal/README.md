# Internal

This directory contains the internal implementation of the project. It is not intended to be used by the end user.
This directory only contains yo test models needed in the test cases.

## Structure

The internal directory is structured as follows:

```text
internal/
    ├── sql/ // Contains the DDL to generate the test model using yo tool
    │   ├── ddl.go
    ├── user/ // Contains the test model and the generated code for the test model
    │   ├── user.yo.go
    │   ├── yo_db.yo.go
```
