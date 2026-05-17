import { readFileSync } from "node:fs";
import { resolve } from "node:path";
import YAML from "yaml";
import { generateApiTypes } from "./api-type-generator.mjs";

const openapiPath = resolve("../docs/openapi.yaml");
const outputPath = resolve("src/api/generated.ts");
const document = YAML.parse(readFileSync(openapiPath, "utf8"));
const expected = generateApiTypes(document);
const actual = readFileSync(outputPath, "utf8");

if (actual !== expected) {
  console.error("src/api/generated.ts is out of date. Run npm run generate:api-types.");
  process.exit(1);
}
