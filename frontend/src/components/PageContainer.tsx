import { motion } from "framer-motion";
import React, { forwardRef } from "react";

import { LAYOUT } from "@/constants/layout";
import { cn } from "@/utils/cn";

interface PageContainerProps extends React.HTMLAttributes<HTMLDivElement> {
  animate?: boolean;
  children: React.ReactNode;
}

export const PageContainer = forwardRef<HTMLDivElement, PageContainerProps>(
  ({ animate = true, children, className, ...props }, ref) => {
    const styles = cn(LAYOUT.PAGE.CONTAINER, className);

    if (!animate) {
      return (
        <div ref={ref} className={styles} {...props}>
          {children}
        </div>
      );
    }

    return (
      <motion.div
        ref={ref}
        animate={LAYOUT.PAGE.ANIMATION.animate}
        className={styles}
        initial={LAYOUT.PAGE.ANIMATION.initial}
        transition={LAYOUT.PAGE.ANIMATION.transition}
        {...(props as any)}
      >
        {children}
      </motion.div>
    );
  },
);

PageContainer.displayName = "PageContainer";
