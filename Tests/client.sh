#!/bin/bash

gnutls-cli --x509certfile cert.pem \
	   --x509keyfile key.pem \
	   --insecure \
	   --port $1 \
	   localhost

exit $?
