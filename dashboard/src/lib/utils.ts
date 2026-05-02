import { type ClassValue, clsx } from "clsx";
import { twMerge } from "tailwind-merge";

/** دمج أسماء CSS مع دعم Tailwind merge */
export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs));
}
