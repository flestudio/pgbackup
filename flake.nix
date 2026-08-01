rec {
  description = "PostgreSQL backup tool for flestudio with S3 upload and Discord notifications";

  inputs.nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";

  outputs =
    { self, nixpkgs }:
    let
      systems = [
        "x86_64-linux"
        "aarch64-linux"
        "x86_64-darwin"
        "aarch64-darwin"
      ];
      forAllSystems = f: nixpkgs.lib.genAttrs systems (system: f nixpkgs.legacyPackages.${system});
      version = self.shortRev or self.dirtyShortRev or "dev";
    in
    {
      packages = forAllSystems (pkgs: rec {
        default = pgbackup;
        pgbackup = pkgs.buildGoModule {
          pname = "pgbackup";
          inherit version;
          src = ./.;
          vendorHash = "sha256-hQT9akOg1sJbwCkfkXzsNuvqnJugnGZgcHnU2vA87hM=";
          subPackages = [ "cmd/pgbackup" ];
          ldflags = [
            "-s"
            "-w"
            "-X main.version=${version}"
          ];
          env.CGO_ENABLED = 0;
          # pg_dump is a runtime dependency; wrap the binary to put it on PATH.
          nativeBuildInputs = [ pkgs.makeWrapper ];
          postInstall = ''
            wrapProgram $out/bin/pgbackup \
              --prefix PATH : ${nixpkgs.lib.makeBinPath [ pkgs.postgresql ]}
          '';
          meta = {
            inherit description;
            homepage = "https://github.com/flestudio/pgbackup";
            license = nixpkgs.lib.licenses.mit;
            mainProgram = "pgbackup";
          };
        };
      });

      devShells = forAllSystems (pkgs: {
        default = pkgs.mkShell {
          packages = with pkgs; [
            go
            gopls
            golangci-lint
            goreleaser
            postgresql # pg_dump, for local testing
            just
          ];
        };
      });
    };
}
