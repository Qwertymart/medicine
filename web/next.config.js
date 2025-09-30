/** @type {import('next').NextConfig} */
const nextConfig = {
  webpack: (config) => {
    config.module.rules.push({
      test: /\.svg$/i,
      issuer: /\.[jt]sx?$/,
      use: ['@svgr/webpack'],
    });
    return config;
  },
  eslint: {
    // пропустить ошибки ESLint при сборке
    ignoreDuringBuilds: true,
  },
  typescript: {
    // пропустить ошибки TypeScript при сборке
    ignoreBuildErrors: true,
  },
};

module.exports = nextConfig;
