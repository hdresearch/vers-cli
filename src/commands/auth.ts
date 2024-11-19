import * as fs from "fs";
import * as os from "os";
import * as path from "path";
// eslint-disable-next-line @typescript-eslint/no-var-requires
const { open } = require("out-url");
import { logger } from "../logger";
import yargs, { Argv } from "yargs";
import { hideBin } from "yargs/helpers";
import { loadConfigFile } from "../lib";

const writeToTudorrc = (key: string, value: string): void => {
  const tudorConfigPath = path.resolve(os.homedir(), ".tudorrc");
  let tudorrc = {};
  try {
    const tudorrcContent = fs.readFileSync(tudorConfigPath, "utf8");
    tudorrc = JSON.parse(tudorrcContent);
  } catch (error) {
    // File does not exist or is not valid JSON
  }
  tudorrc = { ...tudorrc, [key]: value };
  fs.writeFileSync(tudorConfigPath, JSON.stringify(tudorrc));
};

const handleHDRApiKey = async (homeConfig: { hdrApiKey: string }) => {
  const { hdrApiKey } = homeConfig;

  if (!hdrApiKey) {
    logger.warn("No HDR API key found in ~/.tudorrc.");
    const create = await logger.prompt("Would you like to open dashboard.hdr.is?", {
      type: "confirm",
    });

    if (create) {
      open("https://dashboard.hdr.is");
    }
    const apiKey = await logger.prompt("Please enter your HDR API key, or press enter to skip.", {
      type: "text",
    });
    writeToTudorrc("hdrApiKey", apiKey);
  } else {
    logger.log("HDR API key found in ~/.tudorrc.");
    const deleteKey = await logger.prompt("Would you like to delete it?", {
      type: "confirm",
    });

    if (deleteKey) {
      writeToTudorrc("hdrApiKey", "");
      logger.success("HDR API key deleted.");
    }
  }
};

const hdrCommand = {
  command: "hdr",
  describe: "Manage HDR API key configuration",
  handler: async () => {
    const homeConfig = loadConfigFile();
    await handleHDRApiKey(homeConfig as { hdrApiKey: string });
  },
};

const showCommand = {
  command: "show",
  describe: "Show current configuration",
  handler: () => {
    const config = loadConfigFile();
    logger.log(JSON.stringify(config, null, 2));
  },
};

export const command = "config";
export const describe = "Authorize tudor with your High Dimensional Research API key";
export const builder = (yargs: Argv) => {
  return yargs
    .command(hdrCommand)
    .command(showCommand)
    .middleware((argv) => {
      if (argv._.length === 0) {
        yargs.showHelp();
        process.exit(1);
      }
    })
    .demandCommand(1, "You need at least one command before moving on");
};
export const handler = () => {}; // Empty handler since we handle everything in builder

yargs(hideBin(process.argv))
  .scriptName("tudor")
  .command({
    command,
    describe,
    builder,
    handler,
  })
  .demandCommand(1, "You need at least one command before moving on")
  .help();
