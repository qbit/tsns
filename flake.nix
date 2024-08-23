{
  description = "tsns: self-contained name server for your tailnet";

  inputs.nixpkgs.url = "nixpkgs/nixos-unstable";

  outputs =
    { self
    , nixpkgs
    ,
    }:
    let
      supportedSystems = [ "x86_64-linux" "x86_64-darwin" "aarch64-linux" "aarch64-darwin" ];
      forAllSystems = nixpkgs.lib.genAttrs supportedSystems;
      nixpkgsFor = forAllSystems (system: import nixpkgs { inherit system; });
    in
    {
      overlay = _: prev: { inherit (self.packages.${prev.system}) tsns; };

      packages = forAllSystems (system:
        let
          pkgs = nixpkgsFor.${system};
        in
        {
          tsns = pkgs.buildGoModule {
            pname = "tsns";
            version = "v0.0.0";
            src = ./.;

            vendorHash = "sha256-qhzjJfLurlxmWVrmmH7LIPMNNQk6sS/jWptdi6ga9Rk=";
          };
        });

      defaultPackage = forAllSystems (system: self.packages.${system}.tsns);
      nixosModule.default = import ./module.nix;
      devShells = forAllSystems (system:
        let
          pkgs = nixpkgsFor.${system};
        in
        {
          default = pkgs.mkShell {
            shellHook = ''
              PS1='\u@\h:\@; '
              nix run github:qbit/xin#flake-warn
              echo "Go `${pkgs.go}/bin/go version`"
            '';
            nativeBuildInputs = with pkgs; [ git go gopls go-tools ];
          };
        });
    };
}
