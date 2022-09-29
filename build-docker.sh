#! /bin/bash

IMAGE_NAME=robocar-steering-tflite-edgetpu
TAG=$(git describe)
FULL_IMAGE_NAME=docker.io/cyrilix/${IMAGE_NAME}:${TAG}
BINARY=rc-steering
TFLITE_VERSION=2.9.1
GOLANG_VERSION=1.19

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
  LIB_ARCH=x86_64-linux-gnu
  LIB_FLAGS="-L /usr/local/lib/${LIB_ARCH} \
              -L/usr/local/lib/${LIB_ARCH}/absl/base -labsl_base -labsl_throw_delegate -labsl_raw_logging_internal -labsl_spinlock_wait -labsl_malloc_internal -labsl_log_severity \
              -L/usr/local/lib/${LIB_ARCH}/absl/status -labsl_status \
              -L/usr/local/lib/${LIB_ARCH}/absl/hash -labsl_hash -labsl_city -labsl_low_level_hash \
              -L/usr/local/lib/${LIB_ARCH}/absl/flags -labsl_flags -labsl_flags_internal -labsl_flags_marshalling -labsl_flags_reflection -labsl_flags_config -labsl_flags_program_name -labsl_flags_private_handle_accessor -labsl_flags_commandlineflag -labsl_flags_commandlineflag_internal\
              -L/usr/local/lib/${LIB_ARCH}/absl/types -labsl_bad_variant_access -labsl_bad_optional_access -labsl_bad_any_cast_impl \
              -L/usr/local/lib/${LIB_ARCH}/absl/strings -labsl_strings -labsl_str_format_internal -labsl_cord -labsl_cordz_info -labsl_cord_internal -labsl_cordz_functions -labsl_cordz_handle -labsl_strings_internal \
              -L/usr/local/lib/${LIB_ARCH}/absl/time -labsl_time -labsl_time_zone -labsl_civil_time \
              -L/usr/local/lib/${LIB_ARCH}/absl/numeric -labsl_int128 \
              -L/usr/local/lib/${LIB_ARCH}/absl/synchronization -labsl_synchronization -labsl_graphcycles_internal\
              -L/usr/local/lib/${LIB_ARCH}/absl/debugging -labsl_stacktrace -labsl_symbolize -labsl_debugging_internal -labsl_demangle_internal \
              -L/usr/local/lib/${LIB_ARCH}/absl/profiling -labsl_exponential_biased \
              -L/usr/local/lib/${LIB_ARCH}/absl/container -labsl_raw_hash_set -labsl_hashtablez_sampler"
  buildah run \
      --env CGO_ENABLED=1 \
      --env CC=gcc \
      --env CXX=g++ \
      --env GOOS=linux \
      --env GOARCH=amd64 \
      --env GOARM=${GOARM} \
      --env CGO_CPPFLAGS="-I/usr/local/include" \
      --env CGO_LDFLAGS="${LIB_FLAGS}" \
      $containerName \
    go build -a -o rc-steering.amd64 ./cmd/rc-steering
      #--env CGO_CXXFLAGS="--std=c++1z" \

  printf "Compile for linux/arm/v7\n"
  LIB_ARCH=arm-linux-gnueabihf
  LIB_FLAGS="-L /usr/local/lib/${LIB_ARCH} \
              -L/usr/local/lib/${LIB_ARCH}/absl/base -labsl_base -labsl_throw_delegate -labsl_raw_logging_internal -labsl_spinlock_wait -labsl_malloc_internal -labsl_log_severity \
              -L/usr/local/lib/${LIB_ARCH}/absl/status -labsl_status \
              -L/usr/local/lib/${LIB_ARCH}/absl/hash -labsl_hash -labsl_city -labsl_low_level_hash \
              -L/usr/local/lib/${LIB_ARCH}/absl/flags -labsl_flags -labsl_flags_internal -labsl_flags_marshalling -labsl_flags_reflection -labsl_flags_config -labsl_flags_program_name -labsl_flags_private_handle_accessor -labsl_flags_commandlineflag -labsl_flags_commandlineflag_internal\
              -L/usr/local/lib/${LIB_ARCH}/absl/types -labsl_bad_variant_access -labsl_bad_optional_access -labsl_bad_any_cast_impl \
              -L/usr/local/lib/${LIB_ARCH}/absl/strings -labsl_strings -labsl_str_format_internal -labsl_cord -labsl_cordz_info -labsl_cord_internal -labsl_cordz_functions -labsl_cordz_handle -labsl_strings_internal \
              -L/usr/local/lib/${LIB_ARCH}/absl/time -labsl_time -labsl_time_zone -labsl_civil_time \
              -L/usr/local/lib/${LIB_ARCH}/absl/numeric -labsl_int128 \
              -L/usr/local/lib/${LIB_ARCH}/absl/synchronization -labsl_synchronization -labsl_graphcycles_internal\
              -L/usr/local/lib/${LIB_ARCH}/absl/debugging -labsl_stacktrace -labsl_symbolize -labsl_debugging_internal -labsl_demangle_internal \
              -L/usr/local/lib/${LIB_ARCH}/absl/profiling -labsl_exponential_biased \
              -L/usr/local/lib/${LIB_ARCH}/absl/container -labsl_raw_hash_set -labsl_hashtablez_sampler"
  buildah run \
      --env CGO_ENABLED=1 \
      --env CC=arm-linux-gnueabihf-gcc \
      --env CXX=arm-linux-gnueabihf-g++ \
      --env GOOS=linux \
      --env GOARCH=arm \
      --env GOARM=7 \
      --env CGO_CPPFLAGS="-I/usr/local/include" \
      --env CGO_LDFLAGS="${LIB_FLAGS}" \
      $containerName \
    go build -a -o rc-steering.armhf ./cmd/rc-steering

  printf "Compile for linux/arm64\n"
  LIB_ARCH=aarch64-linux-gnu
  LIB_FLAGS="-L /usr/local/lib/${LIB_ARCH} \
              -L/usr/local/lib/${LIB_ARCH}/absl/base -labsl_base -labsl_throw_delegate -labsl_raw_logging_internal -labsl_spinlock_wait -labsl_malloc_internal -labsl_log_severity \
              -L/usr/local/lib/${LIB_ARCH}/absl/status -labsl_status \
              -L/usr/local/lib/${LIB_ARCH}/absl/hash -labsl_hash -labsl_city -labsl_low_level_hash \
              -L/usr/local/lib/${LIB_ARCH}/absl/flags -labsl_flags -labsl_flags_internal -labsl_flags_marshalling -labsl_flags_reflection -labsl_flags_config -labsl_flags_program_name -labsl_flags_private_handle_accessor -labsl_flags_commandlineflag -labsl_flags_commandlineflag_internal\
              -L/usr/local/lib/${LIB_ARCH}/absl/types -labsl_bad_variant_access -labsl_bad_optional_access -labsl_bad_any_cast_impl \
              -L/usr/local/lib/${LIB_ARCH}/absl/strings -labsl_strings -labsl_str_format_internal -labsl_cord -labsl_cordz_info -labsl_cord_internal -labsl_cordz_functions -labsl_cordz_handle -labsl_strings_internal \
              -L/usr/local/lib/${LIB_ARCH}/absl/time -labsl_time -labsl_time_zone -labsl_civil_time \
              -L/usr/local/lib/${LIB_ARCH}/absl/numeric -labsl_int128 \
              -L/usr/local/lib/${LIB_ARCH}/absl/synchronization -labsl_synchronization -labsl_graphcycles_internal\
              -L/usr/local/lib/${LIB_ARCH}/absl/debugging -labsl_stacktrace -labsl_symbolize -labsl_debugging_internal -labsl_demangle_internal \
              -L/usr/local/lib/${LIB_ARCH}/absl/profiling -labsl_exponential_biased \
              -L/usr/local/lib/${LIB_ARCH}/absl/container -labsl_raw_hash_set -labsl_hashtablez_sampler"
  buildah run \
      --env CGO_ENABLED=1 \
      --env CC=aarch64-linux-gnu-gcc \
      --env CXX=aarch64-linux-gnu-g++ \
      --env GOOS=linux \
      --env GOARCH=arm64 \
      --env CGO_CPPFLAGS="-I/usr/local/include" \
      --env CGO_LDFLAGS="${LIB_FLAGS}" \
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
