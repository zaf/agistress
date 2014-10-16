#!/usr/bin/env perl

# Calculate average number of sessions and duration in FastAGI benchmark results
#
# Copyright (C) 2014, Lefteris Zafiris <zaf.000@gmail.com>
#
# This program is free software, distributed under the terms of
# the GNU General Public License Version 3. See the LICENSE file
# at the top of the source tree.

use strict;
use warnings;
use autodie;

if (!$ARGV[0] or $ARGV[0] eq '-h' or $ARGV[0] eq '--help') {
	print "Calculate average values from FastAGI benchmark log files\nUsage: $0 [FILES]\n";
	exit;
}

my @file_list = @ARGV;

foreach my $file (@file_list) {
	my $runs = 0;
	my $active  = 0;
	my $duration = 0;
	my ($min_a, $max_a);
	my ($min_d, $max_d);
	open(my $csvfile, "<", "$file");
	while (<$csvfile>) {
		if (/^\d+,(\d+),(\d+)$/) {
			$active+=$1;
			$min_a = $1 if (!$min_a or $1 < $min_a);
			$max_a = $1 if (!$max_a or $1 > $max_a);
			$duration+=$2;
			$min_d = $2 if (!$min_d or $2 < $min_d);
			$max_d = $2 if (!$max_d or $2 > $max_d);
			$runs++;
		}
	}
	print "\nResults for $file:\n";
	if (!$runs) {
		print "No data found.\n";
	} else {
		print "Average values after $runs runs:\n";
		print "Active Sessions (avr/min/max):  " . int($active/$runs) . " / ", $min_a . " / ", $max_a . "\n";
		print "Session Duration (avr/min/max): " . int($duration/$runs) . " / ", $min_d . " / ", $max_d. "\n";
	}
	close $csvfile;
}
