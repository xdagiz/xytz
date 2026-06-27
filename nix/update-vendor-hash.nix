{ writeShellApplication, gnugrep }:

writeShellApplication {
  name = "update-vendor-hash";
  runtimeInputs = [ gnugrep ];
  text = ''
    HASH_FILE="nix/vendor-hash"

    if BUILD_OUTPUT=$(nix build .#default --no-link 2>&1); then
      echo "vendor-hash is up to date"
      exit 0
    fi

    NEW_HASH=$(echo "$BUILD_OUTPUT" | grep -m1 -oP '^\s+got:\s*\K.*')

    if [[ -z "$NEW_HASH" ]]; then
      echo "Build failed without hash mismatch:"
      echo "$BUILD_OUTPUT"
      exit 1
    fi

    echo "$NEW_HASH" > "$HASH_FILE"
    echo "Updated $HASH_FILE to $NEW_HASH"
  '';
}
