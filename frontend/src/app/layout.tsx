import type { Metadata } from "next";
import { Space_Grotesk, JetBrains_Mono, Inter } from "next/font/google";
import { TooltipProvider } from "@/components/ui/tooltip";
import { Toaster } from "@/components/ui/sonner";
import { AppProviders } from "@/components/providers/app-providers";
import "./globals.css";

/**
 * Type system (frozen, see theme.css):
 *  - Space Grotesk 700/900 — display / headings / logo (uppercase)
 *  - JetBrains Mono 400/600/700 — UI / code / labels / mascot voice
 *  - Inter 400/600 — body copy
 */
const spaceGrotesk = Space_Grotesk({
  variable: "--font-space-grotesk",
  subsets: ["latin"],
  weight: ["700"],
  display: "swap",
});

const jetbrainsMono = JetBrains_Mono({
  variable: "--font-jetbrains-mono",
  subsets: ["latin"],
  weight: ["400", "600", "700"],
  display: "swap",
});

const inter = Inter({
  variable: "--font-inter",
  subsets: ["latin"],
  weight: ["400", "600"],
  display: "swap",
});

export const metadata: Metadata = {
  title: "Transverse — Master DSA",
  description:
    "Transverse is an adaptive DSA & competitive programming tutor. Evaluate your skill, learn a roadmap tailored to your gaps, and practice against a heuristic engine that adjusts to you.",
};

export default function RootLayout({ children }: LayoutProps<"/">) {
  return (
    <html
      lang="en"
      className={`${spaceGrotesk.variable} ${jetbrainsMono.variable} ${inter.variable} h-full antialiased`}
    >
      <body className="min-h-full flex flex-col bg-tv-bg-page">
        <AppProviders>
          <TooltipProvider>
            {children}
            <Toaster
              theme="dark"
              toastOptions={{
                classNames: {
                  toast: "!bg-tv-surface !border-tv-border !text-tv-text-hi",
                },
              }}
            />
          </TooltipProvider>
        </AppProviders>
      </body>
    </html>
  );
}
