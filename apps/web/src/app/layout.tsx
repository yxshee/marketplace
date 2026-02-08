import type { Metadata } from "next";
import Link from "next/link";
import "./globals.css";

export const metadata: Metadata = {
  title: "Marketplace",
  description: "Gumroad-inspired multi-vendor marketplace",
};

const navLinks = [
  { href: "/", label: "Buyer" },
  { href: "/search", label: "Search" },
  { href: "/vendor", label: "Vendor" },
  { href: "/admin", label: "Admin" },
];

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <html lang="en">
      <body>
        <header className="border-b border-border bg-white/90 backdrop-blur">
          <div className="mx-auto flex max-w-6xl items-center justify-between gap-4 px-4 py-4 sm:px-6">
            <Link className="font-display text-base font-semibold" href="/">
              marketplace-gumroad-inspired
            </Link>
            <nav aria-label="Primary" className="flex items-center gap-2 text-sm text-muted sm:gap-3">
              {navLinks.map((item) => (
                <Link
                  className="rounded-sm px-2 py-1 transition-colors hover:bg-black/5 hover:text-ink"
                  href={item.href}
                  key={item.href}
                >
                  {item.label}
                </Link>
              ))}
            </nav>
          </div>
        </header>
        <main className="mx-auto max-w-6xl px-4 py-8 sm:px-6 sm:py-12">{children}</main>
      </body>
    </html>
  );
}
