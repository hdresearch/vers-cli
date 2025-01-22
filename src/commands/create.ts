import { loadConfigFile } from "src/lib";
import { logger } from "../logger";
import { Argv } from "yargs";
import { yellow } from "picocolors";

export const command = "create";
export const describe = "Creates new resource on the HDR platform";
export const aliases = ["c"];

export const builder = (yargs: Argv) => yargs;

export async function handler() {
  const config = loadConfigFile();
  if (!config.hdrApiKey) {
    logger.error("No HDR API key found in ~/.versrc.");
    logger.info(yellow("Please run 'vers config hdr' to configure the HDR API key."));
    return;
  }
  // hit create endpoint, return location / open browser directly for configuration
}
