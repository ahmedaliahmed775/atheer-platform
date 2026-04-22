import { type ClassValue, clsx } from "clsx";
import { twMerge } from "tailwind-merge";

export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs));
}

// Ensure the switch URL uses the environment variable or defaults to localhost 8080
export const SWITCH_API_URL = process.env.NEXT_PUBLIC_SWITCH_API_URL || "http://localhost:8080/api/v2";

export async function fetchApi(path: string, options?: RequestInit) {
  const url = `${SWITCH_API_URL}${path}`;
  const res = await fetch(url, {
    ...options,
    headers: {
      "Content-Type": "application/json",
      ...(options?.headers || {}),
    },
  });
  
  if (!res.ok) {
    throw new Error(`API error: ${res.status} ${res.statusText}`);
  }
  
  // Handlers may return empty body for 200 OK without content
  const text = await res.text();
  if (!text) return {};
  
  try {
    return JSON.parse(text);
  } catch (e) {
    return text; // fallback for non-json
  }
}
