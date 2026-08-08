import { QRCodeSVG } from "qrcode.react";

// A scannable stand-in for typing a 40+ character random secret by hand
// into a datalogger's tiny keypad/screen — this doesn't auto-configure
// any specific hardware (no datalogger integration exists to do that),
// it just puts the exact connection details in front of an installer's
// phone camera so they can read/copy them accurately instead of
// transcribing a long string character by character and risking a typo.
//
// Plain labeled text, not JSON — deliberately: a generic phone camera's
// QR preview shows the raw decoded string, and a labeled multi-line
// block reads far more usably there than a minified JSON blob. If a
// dedicated provisioning app is ever built, encoding structured JSON
// instead would be the one-line change to make here.
export function DeviceQRCode({ deviceId, secret }: { deviceId: string; secret: string }) {
  const brokerURL = import.meta.env.VITE_MQTT_PUBLIC_BROKER_URL as string | undefined;

  const lines = [
    "Clean Energy Analytics — Device Setup",
    ...(brokerURL ? [`Broker: ${brokerURL}`] : []),
    `Device ID: ${deviceId}`,
    `Secret: ${secret}`,
  ];
  const payload = lines.join("\n");

  return (
    <div className="flex flex-col items-center gap-3 glass-card rounded-2xl p-5">
      <div className="bg-white p-3 rounded-xl">
        <QRCodeSVG value={payload} size={160} level="M" />
      </div>
      <p className="text-[11px] text-on-surface-variant text-center max-w-[220px]">
        Scan with a phone camera to read the broker address, device ID, and secret without retyping them by hand.
      </p>
      {!brokerURL && (
        <p className="text-[10px] text-secondary text-center max-w-[220px]">
          Broker address isn't configured (VITE_MQTT_PUBLIC_BROKER_URL) — only device ID and secret are included.
        </p>
      )}
    </div>
  );
}
