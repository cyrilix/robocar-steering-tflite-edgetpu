#! /bin/bash

IMAGE_NAME=robocar-steering-tflite-edgetpu
TAG=$(git describe)
FULL_IMAGE_NAME=docker.io/cyrilix/${IMAGE_NAME}:${TAG}
BINARY=rc-steering
TFLITE_VERSION=2.6.0
GOLANG_VERSION=1.17

GOTAGS="-tags netgo"
BUILDER_CONTAINER="${IMAGE_NAME}-builder"

image_build_binaries(){
  local containerName="$BUILDER_CONTAINER"

  buildah --os "linux" --arch "amd64" --name "$containerName" from "docker.io/cyrilix/tflite-builder:v${TFLITE_VERSION}"

  printf "Copy go binary\n"
  buildah copy --from=docker.io/library/golang:${GOLANG_VERSION} "$containerName" /usr/local/go /usr/local/go
  buildah config \
    --env GOPATH=/go \
    --env PATH=/usr/local/go/bin:/go/bin:/usr/local/go/bin:/usr/local/bin:/usr/bin:/bin \
    $containerName

  buildah config --workingdir /src $containerName

  printf "Copy go sources\n"
  buildah add $containerName go.mod .
  buildah add $containerName go.sum .
  buildah add $containerName vendor/ vendor
  buildah add $containerName pkg/ pkg
  buildah add $containerName cmd/ cmd

  buildah run $containerName ln -s /usr/lib/arm-linux-gnueabihf/libedgetpu.so.1 /usr/lib/arm-linux-gnueabihf/libedgetpu.so
  buildah run $containerName ln -s /usr/lib/aarch64-linux-gnu/libedgetpu.so.1 /usr/lib/aarch64-linux-gnu/libedgetpu.so

  printf "Compile for linux/amd64\n"
  buildah run \
      --env CGO_ENABLED=1 \
      --env CC=gcc \
      --env CXX=g++ \
      --env GOOS=linux \
      --env GOARCH=amd64 \
      --env GOARM=${GOARM} \
      --env CGO_CPPFLAGS="-I/usr/local/include" \
      --env CGO_LDFLAGS="-L /usr/local/lib/x86_64-linux-gnu -L /usr/lib/x86_64-linux-gnu" \
      $containerName \
    go build -a -o rc-steering.amd64 ./cmd/rc-steering
      #--env CGO_CXXFLAGS="--std=c++1z" \

  printf "Compile for linux/arm/v7\n"
  buildah run \
      --env CGO_ENABLED=1 \
      --env CC=arm-linux-gnueabihf-gcc \
      --env CXX=arm-linux-gnueabihf-g++ \
      --env GOOS=linux \
      --env GOARCH=arm \
      --env GOARM=7 \
      --env CGO_CPPFLAGS="-I/usr/local/include" \
      --env CGO_LDFLAGS="-L /usr/lib/arm-linux-gnueabihf -L /usr/local/lib/arm-linux-gnueabihf" \
      $containerName \
    go build -a -o rc-steering.armhf ./cmd/rc-steering

  printf "Compile for linux/arm64\n"
  buildah run \
      --env CGO_ENABLED=1 \
      --env CC=aarch64-linux-gnu-gcc \
      --env CXX=aarch64-linux-gnu-g++ \
      --env GOOS=linux \
      --env GOARCH=arm64 \
      --env CGO_CPPFLAGS="-I/usr/local/include" \
      --env CGO_LDFLAGS="-L /usr/lib/aarch64-linux-gnu -L /usr/local/lib/aarch64-linux-gnu" \
      $containerName \
    go build -a -o rc-steering.arm64 ./cmd/rc-steering
}


image_build(){
  local platform=$1


  GOOS=$(echo $platform | cut -f1 -d/) && \
  GOARCH=$(echo $platform | cut -f2 -d/) && \
  GOARM=$(echo $platform | cut -f3 -d/ | sed "s/v//" )
  VARIANT="--variant $(echo $platform | cut -f3 -d/  )"
  if [[ -z "$GOARM" ]] ;
  then
    VARIANT=""
    binary_suffix="$GOARCH"
  else
    binary_suffix="armhf"
  fi


  local containerName="$IMAGE_NAME-$GOARCH$GOARM"


  buildah --os "$GOOS" --arch "$GOARCH" $VARIANT  --name "$containerName" from "docker.io/cyrilix/tflite-runtime:v${TFLITE_VERSION}"
  buildah config --user 1234 "$containerName"
  buildah copy --from="$BUILDER_CONTAINER" "$containerName" "/src/${BINARY}.${binary_suffix}" /go/bin/$BINARY
  buildah config --entrypoint '["/go/bin/'$BINARY'"]' "${containerName}"

  buildah commit --rm --manifest $IMAGE_NAME "${containerName}" "${containerName}"
}

#buildah rmi localhost/$IMAGE_NAME
#buildah manifest rm localhost/${IMAGE_NAME}

image_build_binaries

image_build linux/amd64
image_build linux/arm64
image_build linux/arm/v7


# push image
printf "\n\nPush manifest to %s\n\n" "${FULL_IMAGE_NAME}"
buildah manifest push --rm -f v2s2 "localhost/$IMAGE_NAME" "docker://$FULL_IMAGE_NAME" --all

buildah rm $BUILDER_CONTAINER