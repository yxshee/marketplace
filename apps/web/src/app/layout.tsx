import type { Metadata } from "next";
import Link from "next/link";
import "./globals.css";

export const metadata: Metadata = {
  title: "Marketplace",
  description: "Multi-vendor marketplace platform",
};

const navLinks = [
  { href: "/", label: "Buyer" },
  { href: "/search", label: "Search" },
  { href: "/cart", label: "Cart" },
  { href: "/vendor", label: "Vendor" },
  { href: "/admin", label: "Admin" },
];

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <html lang="en">
      <body data-surface-mode="tinted">
        <header className="border-b border-border bg-surface backdrop-blur">
          <div className="mx-auto flex max-w-6xl items-center justify-between gap-4 px-4 py-4 sm:px-6">
            <Link className="inline-flex items-center gap-2 font-display text-base font-semibold tracking-tight" href="/">
              <span aria-hidden className="brand-dot" />
              <span>marketplace-platform</span>
            </Link>
            <nav aria-label="Primary" className="flex items-center gap-2 text-sm text-muted sm:gap-3">
              {navLinks.map((item) => (
                <Link
                  className="rounded-full border border-transparent px-3 py-1.5 transition-all duration-150 hover:-translate-y-px hover:border-border hover:bg-surface-soft hover:text-ink"
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
