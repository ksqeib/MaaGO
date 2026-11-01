import React from "react";
import logo from "@/assets/maa-logo.png";

export const AppLogo = ({ size = 32 }: { size?: number }) => (
  <img
    alt="MaaGO"
    draggable={false}
    height={size}
    src={logo}
    style={{ width: size, height: size }}
    width={size}
  />
);
