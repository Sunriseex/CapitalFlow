{ pkgs }:

pkgs.buildGoModule {
  pname = "finance-manager";
  version = "0.6.3";

  src = ./.;

  vendorHash = null;

  subPackages = [
    "cmd/payments-manager"
    "cmd/deposit-manager"
  ];

  buildInputs = with pkgs; [
    jq
    bc
  ];

  meta = with pkgs.lib; {
    description = "CLI for managing payments and deposits with Ledger integration";
    license = licenses.mit;
  };
}
