import React from 'react';
import { useTranslation } from 'react-i18next';

export function Logo(props) {
  const { t } = useTranslation();
  return (
    <svg
      {...props}
      id="logo-svg"  
      xmlns="http://www.w3.org/2000/svg"
      viewBox="0 0 400 80"
      role="img"
      aria-label={t("accessibility.logo")}
    >
      <style>{`
        #logo-svg * {
          fill: currentColor !important;
        }
      `}</style>
      <text
        x="20"
        y="50"
        fontSize="36"
        fontFamily="system-ui, -apple-system, sans-serif"
        fontWeight="bold"
        textAnchor="start"
      >
        Stationmaster
      </text>
    </svg>
  );
}
