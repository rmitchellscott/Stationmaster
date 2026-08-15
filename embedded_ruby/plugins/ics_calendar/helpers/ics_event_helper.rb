module IcsEventHelper
  # Placeholder for ICS event processing functionality
  # This module handles event processing for ICS calendars

  def event_should_be_ignored?(event)
    # Check if event should be ignored based on ignore phrases
    return false unless event&.summary

    ignore_phrases = settings['ignore_phrases_exact_match']
    return false unless ignore_phrases.present?

    phrases = line_separated_string_to_array(ignore_phrases)
    phrases.any? { |phrase| event.summary.to_s.strip == phrase.strip }
  end

  def sanitize_description(description)
    # Basic sanitization of event description
    return "" unless description.present?

    description.to_s.strip
  end

  def calname(event)
    # Extract calendar name from event
    # For now, return a generic name
    "Calendar"
  end

  def all_day_event?(event)
    return false unless event&.dtstart

    # Icalendar wraps DTSTART in its own value classes, so a VALUE=DATE property is
    # neither a ::Date nor does it answer #hour: both checks below miss it unless the
    # property is unwrapped first.
    start_at = unwrap_ical_value(event.dtstart)
    return true if start_at.is_a?(Date) && !start_at.is_a?(DateTime)

    # some producers write an all-day event as an explicit midnight-to-midnight span
    return false if event.dtend.blank?

    end_at = unwrap_ical_value(event.dtend)
    [start_at, end_at].all? { |at| at.respond_to?(:hour) && at.hour.zero? && at.min.zero? }
  end

  def unwrap_ical_value(property)
    property.respond_to?(:value) ? property.value : property
  end

  def guaranteed_end_time(event)
    # Ensure event has an end time
    if event.dtend.present?
      event.dtend.in_time_zone(time_zone)
    elsif event.dtstart.present?
      # If no end time, assume 1 hour duration
      event.dtstart.in_time_zone(time_zone) + 1.hour
    else
      now_in_tz + 1.hour
    end
  end

  def formatted_time
    # Get the time format from settings
    settings['time_format'] || "%-l:%M%P"
  end
end