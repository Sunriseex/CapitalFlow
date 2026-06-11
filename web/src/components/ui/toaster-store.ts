import { toast } from "sonner";

type ToastType = "success" | "error" | "info" | "warning" | "loading";

export const toaster = {
  create({
    type = "info",
    title,
    description,
  }: {
    type?: ToastType;
    title: string;
    description?: string;
  }) {
    toast[type](title, { description });
  },
};
