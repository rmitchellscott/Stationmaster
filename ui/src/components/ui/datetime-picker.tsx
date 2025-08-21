"use client"

import * as React from "react"
import { format } from "date-fns"

import { cn } from "@/lib/utils"
import { DatePicker } from "@/components/ui/date-picker"
import { TimePicker } from "@/components/ui/time-picker"
import { Label } from "@/components/ui/label"

interface DateTimePickerProps {
  value?: string // ISO datetime string
  onChange?: (datetime: string) => void
  placeholder?: string
  className?: string
  disabled?: boolean
}

export function DateTimePicker({
  value,
  onChange,
  placeholder = "Select date and time",
  className,
  disabled,
}: DateTimePickerProps) {
  const date = value ? new Date(value) : undefined
  const timeValue = value ? format(new Date(value), "HH:mm") : ""

  const handleDateChange = (newDate: Date | undefined) => {
    if (!newDate) {
      onChange?.(undefined)
      return
    }

    let timeToUse = "00:00"
    if (value) {
      const existingDate = new Date(value)
      timeToUse = format(existingDate, "HH:mm")
    }

    const [hours, minutes] = timeToUse.split(":")
    newDate.setHours(parseInt(hours), parseInt(minutes))
    onChange?.(newDate.toISOString())
  }

  const handleTimeChange = (time: string) => {
    let dateToUse = new Date()
    if (value) {
      dateToUse = new Date(value)
    }

    const [hours, minutes] = time.split(":")
    dateToUse.setHours(parseInt(hours), parseInt(minutes))
    onChange?.(dateToUse.toISOString())
  }

  return (
    <div className={cn("space-y-3", className)}>
      <div>
        <Label className="text-sm font-medium">Date</Label>
        <DatePicker
          date={date}
          onDateChange={handleDateChange}
          placeholder="Pick a date"
          disabled={disabled}
          className="mt-1"
        />
      </div>
      <div>
        <Label className="text-sm font-medium">Time</Label>
        <TimePicker
          value={timeValue}
          onChange={handleTimeChange}
          placeholder="HH:MM"
          disabled={disabled}
          className="mt-1"
        />
      </div>
    </div>
  )
}