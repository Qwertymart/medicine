import type { NextConfig } from "next";

const nextConfig: NextConfig = {
  // swcMinify: true,
  //   output: "export",
  // webpack: (config: any) => {
  //   config.optimization = {
  //     ...config.optimization,
  //     usedExports: true,
  //     minimize: true,
  //   };
  //   return config;
  // },
  images: {
    remotePatterns: [
      {
        protocol: "https",
        hostname: "loremflickr.com",
        port: "",
        pathname: "**",
        search: "",
      },
      {
        protocol: "https",
        hostname: "www.bigorre.org",
        port: "",
        pathname: "**",
        search: "",
      },
      {
        protocol: "https",
        hostname: "i.imgur.com",
        port: "",
        pathname: "**",
        search: "",
      },
      {
        protocol: "http",
        hostname: "127.0.0.1",
        port: "8000",
        pathname: "**",
        search: "",
      },
    ],
  },
};

export default nextConfig;
