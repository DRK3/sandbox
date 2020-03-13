#!/bin/sh
#
# Copyright SecureKey Technologies Inc. All Rights Reserved.
#
# SPDX-License-Identifier: Apache-2.0
#

apk --no-cache add curl jq

# CouchDB takes time to start up, so we will retry if trying to create a profile fails
n=0
maxAttempts=20

until [ $n -ge $maxAttempts ]
do
   responseCreatedTime=$(curl --header "Content-Type: application/json" \
   --request POST \
   --data '{"name":"vc-issuer-2", "uri":"http://vc-issuer-2.com", "signatureType":"Ed25519Signature2018", "did":"did:v1:test:nym:z6MkiJFtehU8FcTu6hAKiBEzzD5LfZHRR9wiiyJCgkbCZ6XN","didPrivateKey":"4Gn9Ttw6Lf3oFBFqJNNdLFMc4mUbbpCYLNSQFGAxaLBXiJ6DuZpJ8qsZGaHqwyBptxJMEB8nFiqHDZ419XHHxudY"}' \
   http://vcs.example.com:8070/profile | jq -r '.created' 2>/dev/null)
   echo "'created' field from profile response is: $responseCreatedTime"

   if [ -n "$responseCreatedTime" ]
   then
      break
   fi
   echo "Invalid 'created' field from response when trying to create a profile (attempt $((n+1))/$maxAttempts)."

   n=$((n+1))
   sleep 5
done

# Once the above call succeeds, we can assume that CouchDB is up.
# If it fails, then this should almost certainly fail too
curl --header "Content-Type: application/json" --request POST --data '{"name":"vc-issuer-1", "uri":"http://vc-issuer-1.com", "signatureType":"Ed25519Signature2018"}' http://vcs.example.com:8070/profile