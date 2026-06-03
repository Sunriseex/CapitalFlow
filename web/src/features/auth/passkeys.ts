import { api } from "../../api/client";

export function browserSupportsPasskeys() {
  return typeof window !== "undefined" && "PublicKeyCredential" in window && Boolean(navigator.credentials);
}

export async function signInWithPasskey() {
  const options = await api.passkeyLoginOptions();
  const credential = await navigator.credentials.get({
    publicKey: publicKeyRequestOptions(options.publicKey),
  });
  if (!credential) {
    throw new Error("Passkey sign in was cancelled");
  }
  return api.passkeyLoginVerify(publicKeyCredentialToJSON(credential as PublicKeyCredential));
}

export async function registerPasskey(password?: string) {
  const options = await api.passkeyRegistrationOptions({ password: password ?? "" });
  const credential = await navigator.credentials.create({
    publicKey: publicKeyCreationOptions(options.publicKey),
  });
  if (!credential) {
    throw new Error("Passkey setup was cancelled");
  }
  return api.passkeyRegistrationVerify(publicKeyCredentialToJSON(credential as PublicKeyCredential));
}

export function passkeyErrorMessage(err: unknown) {
  if (err instanceof DOMException && err.name === "NotAllowedError") {
    return "Passkey operation cancelled";
  }
  if (err instanceof Error && err.message) {
    return err.message;
  }
  return "Passkey operation failed";
}

function publicKeyCreationOptions(options: PublicKeyCredentialCreationOptions): PublicKeyCredentialCreationOptions {
  return {
    ...options,
    challenge: base64URLToBuffer(options.challenge as unknown as string),
    user: {
      ...options.user,
      id: base64URLToBuffer(options.user.id as unknown as string),
    },
    excludeCredentials: options.excludeCredentials?.map((credential) => ({
      ...credential,
      id: base64URLToBuffer(credential.id as unknown as string),
    })),
  };
}

function publicKeyRequestOptions(options: PublicKeyCredentialRequestOptions): PublicKeyCredentialRequestOptions {
  return {
    ...options,
    challenge: base64URLToBuffer(options.challenge as unknown as string),
    allowCredentials: options.allowCredentials?.map((credential) => ({
      ...credential,
      id: base64URLToBuffer(credential.id as unknown as string),
    })),
  };
}

function publicKeyCredentialToJSON(credential: PublicKeyCredential) {
  const response = credential.response;
  const payload: Record<string, unknown> = {
    id: credential.id,
    rawId: bufferToBase64URL(credential.rawId),
    type: credential.type,
    clientExtensionResults: credential.getClientExtensionResults(),
    authenticatorAttachment: credential.authenticatorAttachment,
  };

  if (response instanceof AuthenticatorAttestationResponse) {
    payload.response = {
      attestationObject: bufferToBase64URL(response.attestationObject),
      clientDataJSON: bufferToBase64URL(response.clientDataJSON),
      transports: response.getTransports?.() ?? [],
    };
  } else if (response instanceof AuthenticatorAssertionResponse) {
    payload.response = {
      authenticatorData: bufferToBase64URL(response.authenticatorData),
      clientDataJSON: bufferToBase64URL(response.clientDataJSON),
      signature: bufferToBase64URL(response.signature),
      userHandle: response.userHandle ? bufferToBase64URL(response.userHandle) : undefined,
    };
  }

  return payload;
}

function base64URLToBuffer(value: string) {
  const base64 = value.replace(/-/g, "+").replace(/_/g, "/").padEnd(Math.ceil(value.length / 4) * 4, "=");
  const binary = atob(base64);
  const bytes = new Uint8Array(binary.length);
  for (let index = 0; index < binary.length; index += 1) {
    bytes[index] = binary.charCodeAt(index);
  }
  return bytes.buffer;
}

function bufferToBase64URL(buffer: ArrayBuffer) {
  const bytes = new Uint8Array(buffer);
  let binary = "";
  for (const byte of bytes) {
    binary += String.fromCharCode(byte);
  }
  return btoa(binary).replace(/\+/g, "-").replace(/\//g, "_").replace(/=+$/g, "");
}
