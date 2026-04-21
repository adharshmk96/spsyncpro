import { createCipheriv, createDecipheriv, randomBytes } from "crypto";

const ENCRYPTION_ALGORITHM = "aes-256-gcm";
const IV_LENGTH = 12;
const AUTH_TAG_LENGTH = 16;

function getEncryptionKey(): Buffer {
  const keyHex = process.env.ORG_SECRET_ENCRYPTION_KEY;

  if (!keyHex) {
    throw new Error("ORG_SECRET_ENCRYPTION_KEY is required.");
  }

  if (!/^[0-9a-fA-F]{64}$/.test(keyHex)) {
    throw new Error("ORG_SECRET_ENCRYPTION_KEY must be a 64-character hex string.");
  }

  return Buffer.from(keyHex, "hex");
}

export function encryptOrganizationClientSecret(plainSecret: string): string {
  if (!plainSecret || typeof plainSecret !== "string") {
    throw new Error("A valid client secret is required for encryption.");
  }

  const encryptionKey = getEncryptionKey();
  const iv = randomBytes(IV_LENGTH);
  const cipher = createCipheriv(ENCRYPTION_ALGORITHM, encryptionKey, iv);

  const encryptedBuffer = Buffer.concat([
    cipher.update(plainSecret, "utf8"),
    cipher.final(),
  ]);
  const authTag = cipher.getAuthTag();

  return `${iv.toString("base64")}:${authTag.toString("base64")}:${encryptedBuffer.toString("base64")}`;
}

export function decryptOrganizationClientSecret(encryptedSecret: string): string {
  if (!encryptedSecret || typeof encryptedSecret !== "string") {
    throw new Error("A valid encrypted client secret is required for decryption.");
  }

  const [ivBase64, authTagBase64, encryptedBase64] = encryptedSecret.split(":");

  if (!ivBase64 || !authTagBase64 || !encryptedBase64) {
    throw new Error("Encrypted client secret format is invalid.");
  }

  const iv = Buffer.from(ivBase64, "base64");
  const authTag = Buffer.from(authTagBase64, "base64");
  const encryptedBuffer = Buffer.from(encryptedBase64, "base64");

  if (iv.length !== IV_LENGTH || authTag.length !== AUTH_TAG_LENGTH) {
    throw new Error("Encrypted client secret payload is invalid.");
  }

  const encryptionKey = getEncryptionKey();
  const decipher = createDecipheriv(ENCRYPTION_ALGORITHM, encryptionKey, iv);
  decipher.setAuthTag(authTag);

  const decryptedBuffer = Buffer.concat([
    decipher.update(encryptedBuffer),
    decipher.final(),
  ]);

  return decryptedBuffer.toString("utf8");
}
