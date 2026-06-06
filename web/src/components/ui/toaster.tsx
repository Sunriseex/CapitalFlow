import {
  Toaster as ChakraToaster,
  Toast,
} from "@chakra-ui/react";
import { toaster } from "./toaster-store";

export function Toaster() {
  return (
    <ChakraToaster toaster={toaster}>
      {(toast) => (
        <Toast.Root className={`toast-card toast-${toast.type ?? "info"}`}>
          <Toast.Indicator />
          <div>
            {toast.title ? <Toast.Title>{toast.title}</Toast.Title> : null}
            {toast.description ? <Toast.Description>{toast.description}</Toast.Description> : null}
          </div>
          <Toast.CloseTrigger />
        </Toast.Root>
      )}
    </ChakraToaster>
  );
}
