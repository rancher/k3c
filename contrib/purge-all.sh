#!/bin/bash

k3c rm -fv $(k3c ps -qa)
k3c rmi $(k3c images -qa)
k3c volumes rm $(k3c volumes ls -q)
