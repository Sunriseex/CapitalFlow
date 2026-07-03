import { api } from "../../api/client";

export function browserSupportsPasskeys() {
  return (
    typeof window !== "undefined" &&
    "PublicKeyCredential" in window &&
    Boolean(navigator.credentials)
  );
}

export async function signInWithPasskey() {
  const options = await api.passkeyLoginOptions();
  const credential = await navigator.credentials.get({
    publicKey: publicKeyRequestOptions(options.publicKey),
  });
  if (!credential) {
    throw new Error("Passkey sign in was cancelled");
  }
  return api.passkeyLoginVerify(
    publicKeyCredentialToJSON(credential as PublicKeyCredential),
  );
}

export async function registerPasskey(password: string) {
  const options = await api.passkeyRegistrationOptions({ password });
  const credential = await navigator.credentials.create({
    publicKey: publicKeyCreationOptions(options.publicKey),
  });
  if (!credential) {
    throw new Error("Passkey setup was cancelled");
  }
  return api.passkeyRegistrationVerify(
    publicKeyCredentialToJSON(credential as PublicKeyCredential),
  );
}

type PasskeyErrorMessages = {
  operationCancelled: string;
  operationFailed: string;
};

const defaultPasskeyErrorMessages: PasskeyErrorMessages = {
  operationCancelled: "Passkey operation cancelled",
  operationFailed: "Passkey operation failed",
};

export function passkeyErrorMessage(
  err: unknown,
  messages: PasskeyErrorMessages = defaultPasskeyErrorMessages,
) {
  if (err instanceof DOMException && err.name === "NotAllowedError") {
    return messages.operationCancelled;
  }

  if (
    err instanceof Error &&
    (err.message === "Passkey sign in was cancelled" ||
      err.message === "Passkey setup was cancelled")
  ) {
    return messages.operationCancelled;
  }

  if (err instanceof Error && err.message) {
    return err.message;
  }

  return messages.operationFailed;
}

function publicKeyCreationOptions(
  options: PublicKeyCredentialCreationOptions,
): PublicKeyCredentialCreationOptions {
  return {
    ...options,
    challenge: credentialBuffer(options.challenge),
    user: {
      ...options.user,
      id: credentialBuffer(options.user.id),
    },
    excludeCredentials: options.excludeCredentials?.map((credential) => ({
      ...credential,
      id: credentialBuffer(credential.id),
    })),
  };
}

function publicKeyRequestOptions(
  options: PublicKeyCredentialRequestOptions,
): PublicKeyCredentialRequestOptions {
  return {
    ...options,
    challenge: credentialBuffer(options.challenge),
    allowCredentials: options.allowCredentials?.map((credential) => ({
      ...credential,
      id: credentialBuffer(credential.id),
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
      userHandle: response.userHandle
        ? bufferToBase64URL(response.userHandle)
        : undefined,
    };
  }

  return payload;
}

function credentialBuffer(value: BufferSource | string): ArrayBuffer {
  if (typeof value === "string") {
    return base64URLToBuffer(value);
  }

  if (value instanceof ArrayBuffer) {
    return value.slice(0);
  }

  return new Uint8Array(
    value.buffer,
    value.byteOffset,
    value.byteLength,
  ).slice().buffer;
}

function base64URLToBuffer(value: string) {
  const base64 = value
    .replace(/-/g, "+")
    .replace(/_/g, "/")
    .padEnd(Math.ceil(value.length / 4) * 4, "=");
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
  return btoa(binary)
    .replace(/\+/g, "-")
    .replace(/\//g, "_")
    .replace(/=+$/g, "");
}
