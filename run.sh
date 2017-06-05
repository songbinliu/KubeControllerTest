#!/bin/bash

bin=./controllerTest
kubeMaster=http://10.10.174.116:8080
$bin --v 4 \
                --masterUrl $kubeMaster \
                --namespace "default" \
                --alsologtostderr

#$bin --v 4 \
#                --kubeconfig ./configs/aws.kubeconfig.yaml \
#                --alsologtostderr
