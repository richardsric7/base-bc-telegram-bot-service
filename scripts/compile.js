// Compiles contracts/ERC20Token.sol with solc and writes ABI + bytecode
// JSON artifacts to internal/contracts/erc20/build/ for abigen to consume.
const fs = require("fs");
const path = require("path");
const solc = require("solc");

const root = path.resolve(__dirname, "..");
const srcPath = path.join(root, "contracts", "ERC20Token.sol");
const outDir = path.join(root, "internal", "contracts", "erc20", "build");

const source = fs.readFileSync(srcPath, "utf8");

const input = {
  language: "Solidity",
  sources: {
    "ERC20Token.sol": { content: source },
  },
  settings: {
    optimizer: { enabled: true, runs: 200 },
    outputSelection: {
      "*": {
        "*": ["abi", "evm.bytecode.object"],
      },
    },
  },
};

const output = JSON.parse(solc.compile(JSON.stringify(input)));

if (output.errors) {
  let hasError = false;
  for (const err of output.errors) {
    console.error(err.formattedMessage);
    if (err.severity === "error") hasError = true;
  }
  if (hasError) process.exit(1);
}

const contract = output.contracts["ERC20Token.sol"]["ERC20Token"];

fs.mkdirSync(outDir, { recursive: true });
fs.writeFileSync(
  path.join(outDir, "ERC20Token.abi"),
  JSON.stringify(contract.abi, null, 2)
);
fs.writeFileSync(
  path.join(outDir, "ERC20Token.bin"),
  contract.evm.bytecode.object
);

console.log("Compiled ERC20Token.sol ->", outDir);
