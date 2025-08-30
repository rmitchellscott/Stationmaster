#!/usr/bin/env ruby

require 'bundler/setup'
require 'json'
require 'trmnl/liquid'

# Test with the actual Literature Clock template from the database
class RealTemplateTest
  def initialize
    @environment = TRMNL::Liquid.build_environment
  end
  
  def test_literature_clock
    # Real Literature Clock template from database
    shared_markup = <<~LIQUID
<!-- logic v20250824 -->
{% liquid

assign granularity = trmnl.plugin_settings.custom_fields_values.snap_to_minutes | default: 5
assign granularity_sec = granularity | times: 60
assign current_time = 'today' | date: '%s' | plus: trmnl.user.utc_offset
assign snapped_time = current_time | times: 1.0 | divided_by: granularity_sec | ceil | times: granularity_sec | date: '%Y-%m-%dT%H:%M:%S'
assign key = snapped_time | date: '%H:%M'

assign newline = data | slice: -1
assign datapoints = data | split: newline
assign matches_str = ''

for datapoint in datapoints
  assign parts = datapoint | split: '|'
  if parts[0] == key
    assign matches_str = matches_str | append: forloop.index0 | append: ','
  endif
endfor
assign matches = matches_str | replace_last: ',', '' | split: ','
assign num_matches = matches | size

# missing data for 06:07, 06:18, 08:21, 10:28, 11:46, 12:31, 13:36, 18:44
if num_matches == 0
  assign snapped_time = snapped_time | plus: 60
  assign key = snapped_time | date: '%H:%M'
  for datapoint in datapoints
    assign parts = datapoint | split: '|'
    if parts[0] == key
      assign matches_str = matches_str | append: forloop.index0 | append: ','
    endif
  endfor
  assign matches = matches_str | replace_last: ',', '' | split: ','
  assign num_matches = matches | size
endif

assign rand = current_time | modulo: num_matches
assign quote_i = matches[rand] | plus: 0
assign quote_parts = datapoints[quote_i] | split: '|'
assign quote_keyword = quote_parts[1]
assign quote_in_bold = '<strong>' | append: quote_keyword | append: '</strong>'
assign quote = quote_parts[2] | replace: quote_keyword, quote_in_bold

%}
<style>
  .literature-clock .main-text :is(strong, b) {
    color: black;
    font-weight: 800;
    font-variation-settings: "wght" 800;
  }
</style>
<script>
document.addEventListener("DOMContentLoaded", () => {
  Array.from(document.getElementsByClassName("literature-clock")).forEach(e => {
    const mainTextElm = e.getElementsByClassName("main-text")[0]
    let fontSize = parseFloat(window.getComputedStyle(mainTextElm).fontSize)
    while (e.scrollHeight > e.clientHeight) {
      fontSize --
      if (fontSize < 12) break
      mainTextElm.style.fontSize = `${fontSize}px`
    }
  })
})
</script>

{% template literature_clock %}
<div class="layout layout--center literature-clock">
  <div class="flex flex--col" style="padding: 1.5% 5%">
    <div class="value main-text {{ main_text_class }}" style="font-size: 96px; line-height: 1.25; margin-bottom: 0.25em;">
      {{ quote }}
    </div>
    <div class="{{footer_class}} attribution" style="align-self: flex-end;">
      – <cite>{{ quote_parts[3] }}</cite>, {{ quote_parts[4] }}
    </div>
  </div>
</div>

<div class="title_bar">
  <img class="image" src="https://usetrmnl.com/images/plugins/poetry_today--render.svg">
  <span class="title">Literature Clock</span>
  <span class="instance">{{ quote_parts[0] }}</span>
</div>
{% endtemplate %}
    LIQUID

    layout_template = <<~LIQUID
{% render 'literature_clock',
  quote: quote,
  quote_parts: quote_parts,
  main_text_class: 'text--gray-4 2bit:text--gray-55 4bit:text--gray-55',
  footer_class: 'value--xxsmall text--gray-4 2bit:text--gray-55 4bit:text--gray-55'
%}
    LIQUID

    # Combined template
    combined_template = shared_markup + "\n" + layout_template
    
    # Sample data structure matching what would come from the system
    data = {
      "data" => "00:00|midnight|It was just after midnight|Sample Book|Sample Author\n00:01|minute|one minute past midnight|Another Book|Another Author\n22:30|half past ten|It was half past ten at night|Test Book|Test Author\n",
      "trmnl" => {
        "user" => {
          "utc_offset" => -21600  # -6 hours CST
        },
        "plugin_settings" => {
          "custom_fields_values" => {
            "snap_to_minutes" => 5
          }
        }
      }
    }
    
    puts "Testing Literature Clock Template"
    puts "Template length: #{combined_template.length}"
    puts "Data keys: #{data.keys.inspect}"
    
    begin
      template = Liquid::Template.parse(combined_template, environment: @environment)
      
      if template.errors.any?
        puts "Parse errors: #{template.errors.inspect}"
        return
      end
      
      rendered = template.render(data)
      
      if template.errors.any?
        puts "Render errors: #{template.errors.inspect}"
        return
      end
      
      puts "SUCCESS! Rendered length: #{rendered.length}"
      puts "Preview:"
      puts rendered[0..500]
      puts "\n... (truncated)"
      
    rescue => e
      puts "ERROR: #{e.class}: #{e.message}"
      puts "Backtrace:"
      puts e.backtrace[0..5].join("\n")
    end
  end
end

if __FILE__ == $0
  tester = RealTemplateTest.new
  tester.test_literature_clock
end